module github.com/cairn-app/cairn-reader/cmd/selfhost

go 1.25.0

require (
	github.com/cairn-app/cairn-reader/pkg/auth v0.0.0
	github.com/cairn-app/cairn-reader/pkg/logging v0.0.0
	github.com/cairn-app/cairn-reader/pkg/middleware v0.0.0
	github.com/cairn-app/cairn-reader/services/explore v0.0.0
	github.com/cairn-app/cairn-reader/services/read v0.0.0
	github.com/cairn-app/cairn-reader/services/read/email v0.0.0
	github.com/cairn-app/cairn-reader/services/users v0.0.0
	github.com/go-chi/chi/v5 v5.2.3
	github.com/joho/godotenv v1.5.1
)

require (
	github.com/PuerkitoBio/goquery v1.9.1 // indirect
	github.com/andybalholm/cascadia v1.3.3 // indirect
	github.com/araddon/dateparse v0.0.0-20210429162001-6b43995a97de // indirect
	github.com/asaskevich/govalidator v0.0.0-20200108200545-475eaeb16496 // indirect
	github.com/aymerick/douceur v0.2.0 // indirect
	github.com/cairn-app/cairn-reader v0.0.0 // indirect
	github.com/cairn-app/cairn-reader/pkg/api v0.0.0 // indirect
	github.com/cairn-app/cairn-reader/pkg/models v0.0.0 // indirect
	github.com/cairn-app/cairn-reader/pkg/rss v0.0.0-00010101000000-000000000000 // indirect
	github.com/cenkalti/backoff/v4 v4.3.0 // indirect
	github.com/go-jose/go-jose/v4 v4.1.3 // indirect
	github.com/go-ozzo/ozzo-validation/v4 v4.3.0 // indirect
	github.com/go-shiori/dom v0.0.0-20230515143342-73569d674e1c // indirect
	github.com/go-shiori/go-readability v0.0.0-20251205110129-5db1dc9836f0 // indirect
	github.com/gogs/chardet v0.0.0-20211120154057-b7413eaefb8f // indirect
	github.com/golang-jwt/jwt/v5 v5.3.1 // indirect
	github.com/golang-migrate/migrate/v4 v4.19.1 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/gorilla/css v1.0.1 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/go-retryablehttp v0.7.8 // indirect
	github.com/hashicorp/go-rootcerts v1.0.2 // indirect
	github.com/hashicorp/go-secure-stdlib/parseutil v0.2.0 // indirect
	github.com/hashicorp/go-secure-stdlib/strutil v0.1.2 // indirect
	github.com/hashicorp/go-sockaddr v1.0.7 // indirect
	github.com/hashicorp/hcl v1.0.1-vault-7 // indirect
	github.com/hashicorp/vault/api v1.22.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/pgx/v5 v5.8.0 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/lib/pq v1.10.9 // indirect
	github.com/microcosm-cc/bluemonday v1.0.27 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/mmcdole/gofeed v1.3.0 // indirect
	github.com/mmcdole/goxpp v1.1.1 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/robfig/cron/v3 v3.0.1 // indirect
	github.com/ryanuber/go-glob v1.0.0 // indirect
	github.com/sony/gobreaker v0.5.0 // indirect
	golang.org/x/crypto v0.49.0 // indirect
	golang.org/x/net v0.52.0 // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/text v0.36.0 // indirect
	golang.org/x/time v0.12.0 // indirect
)

// Local replacements for all modules in the monorepo
replace github.com/cairn-app/cairn-reader => ../..

replace github.com/cairn-app/cairn-reader/pkg/api => ../../pkg/api

replace github.com/cairn-app/cairn-reader/pkg/auth => ../../pkg/auth

replace github.com/cairn-app/cairn-reader/pkg/config => ../../pkg/config

replace github.com/cairn-app/cairn-reader/pkg/logging => ../../pkg/logging

replace github.com/cairn-app/cairn-reader/pkg/middleware => ../../pkg/middleware

replace github.com/cairn-app/cairn-reader/pkg/models => ../../pkg/models

replace github.com/cairn-app/cairn-reader/services/users => ../../services/users

replace github.com/cairn-app/cairn-reader/services/explore => ../../services/explore

replace github.com/cairn-app/cairn-reader/services/read => ../../services/read

replace github.com/cairn-app/cairn-reader/services/read/email => ../../services/read/email

replace github.com/cairn-app/cairn-reader/pkg/rss => ../../pkg/rss
