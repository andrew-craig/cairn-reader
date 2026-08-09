module github.com/cairn-app/cairn-reader/services/users

go 1.25.12

require (
	github.com/cairn-app/cairn-reader v0.0.0-20260114203311-1fb2f559a019
	github.com/cairn-app/cairn-reader/pkg/api v0.0.0-00010101000000-000000000000
	github.com/cairn-app/cairn-reader/pkg/auth v0.0.0-00010101000000-000000000000
	github.com/cairn-app/cairn-reader/pkg/logging v0.0.0-00010101000000-000000000000
	github.com/cairn-app/cairn-reader/pkg/middleware v0.0.0-00010101000000-000000000000
	github.com/go-chi/chi/v5 v5.2.3
	github.com/golang-jwt/jwt/v5 v5.3.0
	github.com/golang-migrate/migrate/v4 v4.19.1
	github.com/golang/mock v1.6.0
	github.com/google/uuid v1.6.0
	github.com/hashicorp/vault/api v1.22.0
	github.com/jackc/pgx/v5 v5.10.0
	github.com/joho/godotenv v1.5.1
	github.com/stretchr/testify v1.11.1
	golang.org/x/crypto v0.54.0
)

require (
	github.com/cenkalti/backoff/v4 v4.3.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/go-jose/go-jose/v4 v4.1.1 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/go-retryablehttp v0.7.8 // indirect
	github.com/hashicorp/go-rootcerts v1.0.2 // indirect
	github.com/hashicorp/go-secure-stdlib/parseutil v0.2.0 // indirect
	github.com/hashicorp/go-secure-stdlib/strutil v0.1.2 // indirect
	github.com/hashicorp/go-sockaddr v1.0.7 // indirect
	github.com/hashicorp/hcl v1.0.1-vault-7 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/lib/pq v1.10.9 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/rogpeppe/go-internal v1.14.1 // indirect
	github.com/ryanuber/go-glob v1.0.0 // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	golang.org/x/net v0.57.0 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/text v0.40.0 // indirect
	golang.org/x/time v0.12.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

// Use local version of shared cairn packages
replace github.com/cairn-app/cairn-reader => ../..

replace github.com/cairn-app/cairn-reader/pkg/api => ../../pkg/api

replace github.com/cairn-app/cairn-reader/pkg/auth => ../../pkg/auth

replace github.com/cairn-app/cairn-reader/pkg/logging => ../../pkg/logging

replace github.com/cairn-app/cairn-reader/pkg/middleware => ../../pkg/middleware
