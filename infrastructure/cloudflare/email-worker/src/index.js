// Cairn Email Ingest — Cloudflare Email Worker
//
// Receives mail routed via Cloudflare Email Routing, parses the MIME payload,
// and forwards a normalised JSON envelope to the Email Ingest Service's
// `POST /api/v1/source/email/ingest` endpoint.
//
// Required bindings (see wrangler.toml):
//   - INGEST_API_URL  (var)    Full URL of the ingest endpoint
//   - INGEST_API_KEY  (secret) X-API-Key value the ingest service accepts
//
// SMTP behaviour:
//   - 5xx/network errors  → setReject so the sending MTA retries
//   - 4xx from ingest     → accept silently (the message is malformed for us
//                           and retrying will not help)
//   - 200 with accepted=false (unknown recipient) → setReject so we do not
//                           silently swallow mail for addresses that do not
//                           map to a Cairn user

import PostalMime from "postal-mime";

const MAX_BODY_BYTES = 5 * 1024 * 1024; // mirror the 5 MB content limit

export default {
  async email(message, env, ctx) {
    if (!env.INGEST_API_URL || !env.INGEST_API_KEY) {
      console.error("email-worker: missing INGEST_API_URL or INGEST_API_KEY");
      message.setReject("Mail handler not configured");
      return;
    }

    let parsed;
    try {
      parsed = await PostalMime.parse(message.raw);
    } catch (err) {
      console.error("email-worker: failed to parse MIME", err);
      message.setReject("Could not parse email");
      return;
    }

    const payload = buildIngestPayload(message, parsed);

    if (!payload.html_body && !payload.text_body) {
      console.warn("email-worker: dropping empty email", {
        from: payload.sender,
        to: payload.recipient,
      });
      // Accept silently — an empty body is nothing the user wants saved.
      return;
    }

    let res;
    try {
      res = await fetch(env.INGEST_API_URL, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "X-API-Key": env.INGEST_API_KEY,
        },
        body: JSON.stringify(payload),
      });
    } catch (err) {
      console.error("email-worker: ingest request failed", err);
      message.setReject("Temporary failure, please retry");
      return;
    }

    if (res.status >= 500) {
      const body = await safeText(res);
      console.error("email-worker: ingest 5xx", { status: res.status, body });
      message.setReject("Temporary failure, please retry");
      return;
    }

    if (res.status === 401 || res.status === 403) {
      const body = await safeText(res);
      console.error("email-worker: ingest auth failure", {
        status: res.status,
        body,
      });
      message.setReject("Mail handler not authorised");
      return;
    }

    if (!res.ok) {
      const body = await safeText(res);
      console.error("email-worker: ingest 4xx", { status: res.status, body });
      // Accept so the sender does not keep retrying a malformed envelope.
      return;
    }

    const json = await res.json().catch(() => null);
    if (json?.data?.accepted === false) {
      console.warn("email-worker: ingest rejected recipient", {
        recipient: payload.recipient,
      });
      message.setReject("Recipient address unknown");
    }
  },
};

function buildIngestPayload(message, parsed) {
  const fromAddress = parsed.from?.address || message.from;
  const fromName = parsed.from?.name || undefined;
  const receivedAt = parsed.date ? new Date(parsed.date) : new Date();

  return omitUndefined({
    recipient: message.to,
    sender: fromAddress,
    sender_name: fromName,
    subject: parsed.subject || undefined,
    html_body: truncate(parsed.html, MAX_BODY_BYTES),
    text_body: truncate(parsed.text, MAX_BODY_BYTES),
    received_at: receivedAt.toISOString(),
  });
}

function truncate(value, maxBytes) {
  if (!value) return undefined;
  const encoded = new TextEncoder().encode(value);
  if (encoded.byteLength <= maxBytes) return value;
  return new TextDecoder().decode(encoded.slice(0, maxBytes));
}

function omitUndefined(obj) {
  const out = {};
  for (const [k, v] of Object.entries(obj)) {
    if (v !== undefined && v !== null && v !== "") out[k] = v;
  }
  return out;
}

async function safeText(res) {
  try {
    return await res.text();
  } catch {
    return "";
  }
}
