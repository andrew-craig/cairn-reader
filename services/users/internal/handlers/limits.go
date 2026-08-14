package handlers

// maxRequestBodySize bounds JSON request bodies on auth/user endpoints,
// mirroring the pattern in explore/recommender/internal/api/handlers.go.
// Every body here is a handful of short string fields (email, password,
// tokens), so a single tight cap covers all of them.
const maxRequestBodySize = 16 << 10 // 16KB
