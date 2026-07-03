---
name: webapp-staging-test
description: Run the Cairn web app locally against a remote backend and drive it with a headless browser, using the shared test account. Use when asked to test, verify, or demo a web app change against a staging/remote backend, or to check something in the browser end-to-end.
---

# Testing the web app against a remote backend

Runs `apps/web` locally (Vite dev server) pointed at a remote backend, then
drives it with Chromium via a plain Playwright script. There is no browser
MCP tool in this environment — don't look for one or install one. Use the
globally-installed Playwright package instead (see below).

The backend URL is not committed anywhere in this repo — it points at a
personal environment. **Ask the user for it** if it wasn't given to you.

## 1. Point the dev server at the backend

```bash
cp apps/web/.env.example apps/web/.env   # gitignored, safe to overwrite
# edit apps/web/.env: VITE_API_URL=<backend URL the user gave you>
cd apps/web && npm install               # if node_modules isn't already present
npm run dev -- --port 5173               # leave running in the background
```

The app is then served at `http://localhost:5173`.

## 2. Test account

From `apps/web/CLAUDE.md` — a real seeded account on the default backend.
Log in through the app's normal login form (`#email`, `#password` inputs,
then a `button[type="submit"]`) — there's no separate auth API shortcut.

## 3. Driving the browser

No browser MCP tool is registered in this environment. Chromium is
pre-installed for Playwright at `/opt/pw-browsers/chromium`, and the
`playwright` npm package is installed **globally** (not as a project
dependency) — don't `npm install playwright` into the repo. Run scripts with
`NODE_PATH` pointed at the global modules:

```bash
NODE_PATH=/opt/node22/lib/node_modules node your-script.js
```

Use `launch-browser.js` in this skill directory for a correctly-configured
launch — copy the pattern into your own script rather than requiring this
file directly (paths/ports will differ per task):

```js
const { chromium } = require('playwright');

const browser = await chromium.launch({
  executablePath: '/opt/pw-browsers/chromium',
  args: [
    `--proxy-server=${process.env.HTTPS_PROXY}`,
    '--proxy-bypass-list=localhost;127.0.0.1',
    '--ssl-version-max=tls1.2',
  ],
});
```

### Why these flags

- `--proxy-server` — outbound HTTPS in this environment goes through an agent
  proxy (`$HTTPS_PROXY`); Chromium doesn't pick it up from the environment
  automatically like curl/Node's `fetch` do.
- `--proxy-bypass-list=localhost;127.0.0.1` — sends the Vite dev server traffic
  direct instead of through the tunnel.
- `--ssl-version-max=tls1.2` — **required**. Without this, every HTTPS request
  Chromium makes through the proxy fails with `net::ERR_CONNECTION_RESET`
  (verified: curl and Node's `fetch` succeed through the same proxy at
  TLS 1.3 with no special flags — only Chromium's handshake gets reset).
  Forcing Chromium down to TLS 1.2 fixes it. The proxy CA
  (`/root/.ccr/ca-bundle.crt`) is already trusted by Chromium — no
  `--ignore-certificate-errors` needed.

### Expected noise

Article thumbnails in the Explore/Read feeds are hotlinked from arbitrary
third-party sites seeded by RSS content. Some of those fail through the proxy
with `net::ERR_TUNNEL_CONNECTION_FAILED` — this is normal image-loading noise
unrelated to the app or backend, not a bug to chase.

## 4. Cleanup

Kill the background `npm run dev` process when done. `apps/web/.env` is
gitignored, so it's fine to leave it pointed at the backend for the next run.
