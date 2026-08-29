package handlers

import (
	"errors"
	"net/http"

	"github.com/andrew-craig/cairn-reader/pkg/api"
	apperrors "github.com/andrew-craig/cairn-reader/pkg/errors"
	"github.com/andrew-craig/cairn-reader/services/users/internal/services"
)

// serviceErrorMapping maps a service-layer sentinel error to the HTTP response
// the handlers should produce for it.
type serviceErrorMapping struct {
	sentinel error
	status   int
	code     string
	// message is the client-facing message. When useErrText is true the wrapped
	// error's own Error() string is sent instead — used for sentinels whose
	// message is composed by the service layer (e.g. password-strength detail).
	message    string
	useErrText bool
}

// serviceErrorTable is the single source of truth for sentinel -> HTTP status
// mapping across auth_handler.go and user_handler.go.
//
// It is an ORDERED slice, not a map, and is deliberately so: errors.Is is a
// predicate (it walks the wrap chain), not an equality check, so it cannot be a
// map key. writeServiceError walks this slice top to bottom and the first
// matching row wins. Order therefore encodes precedence between sentinels that
// could both match a single error (e.g. one wrapping another) — put the more
// specific sentinel first. Today none of these overlap, so order is not
// load-bearing yet; keep it specific-to-generic anyway so it stays correct when
// a wrapped sentinel is added.
var serviceErrorTable = []serviceErrorMapping{
	{services.ErrUnauthorized, http.StatusForbidden, api.ErrCodeForbidden, "you can only access your own account", false},
	{services.ErrHybridAccountDeviceLogin, http.StatusForbidden, api.ErrCodeForbidden, "device login not allowed for accounts with email/password", false},
	{services.ErrTokenReused, http.StatusUnauthorized, api.ErrCodeUnauthorized, "token reuse detected - all tokens have been revoked", false},
	{services.ErrRefreshTokenExpired, http.StatusUnauthorized, api.ErrCodeUnauthorized, "refresh token has expired", false},
	{services.ErrIncorrectPassword, http.StatusUnauthorized, api.ErrCodeUnauthorized, "current password is incorrect", false},
	{services.ErrInvalidCredentials, http.StatusUnauthorized, api.ErrCodeUnauthorized, "invalid credentials", false},
	{services.ErrAccountLocked, http.StatusTooManyRequests, api.ErrCodeTooManyRequests, "", true},
	{services.ErrAccountExists, http.StatusConflict, api.ErrCodeConflict, "an account with these credentials already exists", false},
	{services.ErrEmailAlreadyVerified, http.StatusConflict, api.ErrCodeConflict, "email is already verified", false},
	{apperrors.ErrUserNotFound, http.StatusNotFound, api.ErrCodeNotFound, "user not found", false},
	{services.ErrWeakPassword, http.StatusBadRequest, api.ErrCodeValidation, "", true},
	{services.ErrInvalidVerificationToken, http.StatusBadRequest, api.ErrCodeBadRequest, "invalid or expired verification token", false},
	{services.ErrNotMobileAccount, http.StatusBadRequest, api.ErrCodeBadRequest, "account is not a mobile-only account", false},
	{services.ErrNoPassword, http.StatusBadRequest, api.ErrCodeBadRequest, "account does not have a password", false},
	{services.ErrInvalidInput, http.StatusBadRequest, api.ErrCodeBadRequest, "invalid input provided", false},
}

// writeServiceError maps err through serviceErrorTable and writes the matching
// HTTP error response. If no sentinel matches, it writes a 500 with
// fallbackMessage — that is the only path to a 500, so a newly added service
// sentinel that is missing from the table surfaces as an obvious 500 (and is
// caught by TestServiceErrorTable_CoversAllSentinels) rather than being silently
// mishandled by a forgotten per-handler branch.
func writeServiceError(w http.ResponseWriter, err error, fallbackMessage string) {
	for _, m := range serviceErrorTable {
		if errors.Is(err, m.sentinel) {
			message := m.message
			if m.useErrText {
				message = err.Error()
			}
			api.WriteError(w, m.status, m.code, message, nil, "v1")
			return
		}
	}
	api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternal, fallbackMessage, nil, "v1")
}
