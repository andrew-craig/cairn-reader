module github.com/cairn-app/cairn-reader/services/read/email

go 1.24.7

require (
	github.com/cairn-app/cairn-reader/pkg/logging v0.0.0
	github.com/go-chi/chi/v5 v5.2.3
	github.com/golang-migrate/migrate/v4 v4.19.1
	github.com/google/uuid v1.6.0
	github.com/joho/godotenv v1.5.1
	github.com/lib/pq v1.10.9
	github.com/stretchr/testify v1.11.1
)

require (
	github.com/DATA-DOG/go-sqlmock v1.5.2
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/cairn-app/cairn-reader/pkg/logging => ../../../pkg/logging

replace github.com/cairn-app/cairn-reader/pkg/api => ../../../pkg/api

replace github.com/cairn-app/cairn-reader/pkg/config => ../../../pkg/config

replace github.com/cairn-app/cairn-reader/pkg/auth => ../../../pkg/auth
