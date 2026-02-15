# Change Password Endpoint

**Priority:** P2
**Status:** pending

## Problem

The mobile app Account page has a "Change password" feature for authenticated (non-anonymous) users, but the backend User Service has no endpoint to support changing a password. The mobile app currently calls `PUT /api/v1/user/{user_id}/password`, which returns a 404.

## Impact

Users who have upgraded their account (or registered with email) cannot change their password. This is a basic account management feature expected before production launch.

## Current Implementation

The User Service has these account management endpoints:
- `GET /api/v1/user/{user_id}` - Get profile
- `PATCH /api/v1/user/{user_id}` - Update profile (email only)
- `POST /api/v1/user/{user_id}/upgrade` - Upgrade mobile-only to hybrid
- `DELETE /api/v1/user/{user_id}` - Delete account

Password hashing already uses bcrypt (cost 12) via `services/users/internal/auth/password.go`.

## Proposed Solution

Add a `PUT /api/v1/user/{user_id}/password` endpoint that accepts the current password and a new password.

### Request Format
```json
{
  "current_password": "oldpass123",
  "new_password": "newpass456"
}
```

### Response Format
```json
{
  "message": "password changed successfully"
}
```

### Implementation Steps

1. **Repository layer** (`services/users/internal/database/user_repository.go`):
   - Add `UpdatePassword(ctx, userID, passwordHash)` method

2. **Service layer** (`services/users/internal/services/user_service.go`):
   - Add `ChangePassword(ctx, requestingUserID, targetUserID, currentPassword, newPassword)` method
   - Verify `requestingUserID == targetUserID` (authorization)
   - Fetch user, verify current password with bcrypt compare
   - Validate new password strength using existing `auth.ValidatePasswordStrength()`
   - Hash new password and persist

3. **Handler layer** (`services/users/internal/handlers/user_handler.go`):
   - Add `ChangePassword` handler
   - Parse and validate request body
   - Extract user ID from JWT claims
   - Call service layer and return appropriate response

4. **Router** (`services/users/internal/handlers/router.go`):
   - Register `PUT /api/v1/user/:user_id/password` with auth middleware

5. **OpenAPI spec** (`services/users/api/openapi.yaml`):
   - Document the new endpoint

### Error Cases
- 400: Missing fields, weak new password
- 401: Invalid current password, missing/invalid JWT
- 403: User trying to change another user's password
- 404: User not found
- 500: Server error

## Files to Modify

- `services/users/internal/database/user_repository.go`
- `services/users/internal/services/user_service.go`
- `services/users/internal/handlers/user_handler.go`
- `services/users/internal/handlers/router.go`
- `services/users/api/openapi.yaml`

## Testing

- Test successful password change with valid current password
- Test rejection with wrong current password
- Test new password strength validation (min 8 chars)
- Test authorization check (cannot change another user's password)
- Test that the new password works for login after change
- Test error cases (missing fields, user not found)

## Mobile App Integration

The mobile app (`apps/mobile/src/services/auth.ts`) already has `AuthService.changePassword()` calling `PUT /api/v1/user/{user_id}/password` with `{ current_password, new_password }`. Once the backend is implemented, the feature will work end-to-end.
