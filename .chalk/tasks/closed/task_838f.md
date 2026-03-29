---
id: task_838f
title: Implement HTTP handlers and wire router
type: task
status: closed
priority: 1
labels: []
blocked_by: [task_ab9a,task_899b,task_535a]
parent: epic_0c4d
created_at: 2026-03-23T07:18:18Z
updated_at: 2026-03-29T02:40:13Z
---
Implement all three handlers and wire them into the router:

## IngestHandler (handlers/ingest_handler.go)
- POST /api/v1/source/email/ingest
- Protected by API key middleware
- Parse IngestEmailRequest, validate fields
- Call EmailService.IngestEmail
- Return 202 Accepted with IngestEmailResponse{Accepted: true}
- Return 404 if recipient not found (don't leak info — could also be 202)

## AddressHandler (handlers/address_handler.go)
- POST /api/v1/source/email/user/{user_id}/address — create/get address
- GET /api/v1/source/email/user/{user_id}/address — get address
- Protected by JWT middleware
- Enforce user can only access own address (user_id from JWT must match URL param)
- Call AddressService.GetOrCreate / GetByUserID
- Return CreateAddressResponse / GetAddressResponse

## SenderHandler (handlers/sender_handler.go)
- GET /api/v1/source/email/user/{user_id}/senders — list senders
- Protected by JWT middleware
- Enforce user can only access own senders
- Call SenderService.ListByUser
- Return ListSendersResponse

## Router Wiring (api/router.go)
- Mount handlers on the commented-out routes
- Apply API key middleware to ingest route
- Apply JWT middleware to user routes
- Inject services/repos as dependencies

## Tests
- Handler unit tests with mocked services
- Test authorization enforcement (user_id mismatch → 403)
