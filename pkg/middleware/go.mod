module github.com/andrew-craig/cairn-reader/pkg/middleware

go 1.25.12

require (
	github.com/andrew-craig/cairn-reader/pkg/logging v0.0.0-00010101000000-000000000000
	github.com/go-chi/chi/v5 v5.2.3
)

require github.com/google/uuid v1.6.0 // indirect

replace github.com/andrew-craig/cairn-reader/pkg/logging => ../logging
