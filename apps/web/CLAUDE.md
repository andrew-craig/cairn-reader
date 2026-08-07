# Cairn Web App — Agent Notes

## Test account

A reusable account for verifying authenticated flows against the default backend:

- **Email**: `cairn.web.test@seatrain.net`
- **Password**: `CairnWebTest!2026`

Seeded data: 2 RSS feed subscriptions (Hacker News, The Verge) and 1 saved page.

## Backend URL in development

The app defaults to the origin it is served from. Under `npm run dev` that origin
has no API, so set `VITE_API_URL` (copy `.env.example` to `.env`) to point at a
backend, e.g. `VITE_API_URL=http://localhost:8099`.
