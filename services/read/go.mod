module github.com/cairn-app/cairn-reader/services/read

go 1.24.7

require (
	github.com/DATA-DOG/go-sqlmock v1.5.2
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
	github.com/mmcdole/gofeed v1.3.0
	github.com/robfig/cron/v3 v3.0.1
	github.com/sony/gobreaker v0.5.0
	github.com/stretchr/testify v1.11.1
	golang.org/x/net v0.47.0
)

require (
	github.com/PuerkitoBio/goquery v1.8.0 // indirect
	github.com/andybalholm/cascadia v1.3.3 // indirect
	github.com/araddon/dateparse v0.0.0-20210429162001-6b43995a97de // indirect
	github.com/asaskevich/govalidator v0.0.0-20200108200545-475eaeb16496 // indirect
	github.com/aymerick/douceur v0.2.0 // indirect
	github.com/bytedance/sonic v1.14.0 // indirect
	github.com/bytedance/sonic/loader v0.3.0 // indirect
	github.com/cenkalti/backoff/v4 v4.3.0 // indirect
	github.com/cloudwego/base64x v0.1.6 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/gabriel-vasile/mimetype v1.4.8 // indirect
	github.com/gin-contrib/sse v1.1.0 // indirect
	github.com/gin-gonic/gin v1.11.0 // indirect
	github.com/go-jose/go-jose/v4 v4.1.1 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-playground/validator/v10 v10.27.0 // indirect
	github.com/go-shiori/dom v0.0.0-20230515143342-73569d674e1c // indirect
	github.com/goccy/go-json v0.10.2 // indirect
	github.com/goccy/go-yaml v1.18.0 // indirect
	github.com/gogs/chardet v0.0.0-20211120154057-b7413eaefb8f // indirect
	github.com/golang-jwt/jwt/v5 v5.3.0 // indirect
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
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/cpuid/v2 v2.3.0 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/mmcdole/goxpp v1.1.1-0.20240225020742-a0c311522b23 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/pelletier/go-toml/v2 v2.2.4 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/quic-go/qpack v0.5.1 // indirect
	github.com/quic-go/quic-go v0.54.0 // indirect
	github.com/ryanuber/go-glob v1.0.0 // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	github.com/twitchyliquid64/golang-asm v0.15.1 // indirect
	github.com/ugorji/go/codec v1.3.0 // indirect
	go.uber.org/mock v0.5.0 // indirect
	golang.org/x/arch v0.20.0 // indirect
	golang.org/x/crypto v0.45.0 // indirect
	golang.org/x/mod v0.29.0 // indirect
	golang.org/x/sync v0.18.0 // indirect
	golang.org/x/sys v0.38.0 // indirect
	golang.org/x/text v0.31.0 // indirect
	golang.org/x/time v0.12.0 // indirect
	golang.org/x/tools v0.38.0 // indirect
	google.golang.org/protobuf v1.36.9 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/cairn-app/cairn-reader/pkg/logging => ../../pkg/logging

replace github.com/cairn-app/cairn-reader/pkg/api => ../../pkg/api

replace github.com/cairn-app/cairn-reader/pkg/config => ../../pkg/config

replace github.com/cairn-app/cairn-reader/pkg/auth => ../../pkg/auth
