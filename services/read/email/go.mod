module github.com/cairn-app/cairn-reader/services/read/email

go 1.24.7

require (
	github.com/cairn-app/cairn-reader/pkg/api v0.0.0
	github.com/cairn-app/cairn-reader/pkg/auth v0.0.0
	github.com/cairn-app/cairn-reader/pkg/config v0.0.0-00010101000000-000000000000
	github.com/cairn-app/cairn-reader/pkg/logging v0.0.0
	github.com/go-chi/chi/v5 v5.2.3
	github.com/go-ozzo/ozzo-validation/v4 v4.3.0
	github.com/go-shiori/go-readability v0.0.0-20251205110129-5db1dc9836f0
	github.com/golang-migrate/migrate/v4 v4.19.1
	github.com/google/uuid v1.6.0
	github.com/joho/godotenv v1.5.1
	github.com/lib/pq v1.10.9
	github.com/microcosm-cc/bluemonday v1.0.27
	github.com/robfig/cron/v3 v3.0.1
	github.com/sony/gobreaker v0.5.0
	github.com/stretchr/testify v1.11.1
	golang.org/x/net v0.47.0
)

replace github.com/cairn-app/cairn-reader/pkg/logging => ../../../pkg/logging

replace github.com/cairn-app/cairn-reader/pkg/api => ../../../pkg/api

replace github.com/cairn-app/cairn-reader/pkg/config => ../../../pkg/config

replace github.com/cairn-app/cairn-reader/pkg/auth => ../../../pkg/auth
