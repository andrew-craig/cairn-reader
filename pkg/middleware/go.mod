module github.com/cairn-app/cairn-reader/pkg/middleware

go 1.24.0

require github.com/cairn-app/cairn-reader/pkg/logging v0.0.0-00010101000000-000000000000

require (
	github.com/go-chi/chi/v5 v5.2.3 // indirect
	github.com/google/uuid v1.6.0 // indirect
)

replace github.com/cairn-app/cairn-reader/pkg/logging => ../logging
