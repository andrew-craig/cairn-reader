module github.com/andrew-craig/cairn/services/explore

go 1.24.7

require (
	github.com/andrew-craig/cairn v0.0.0-00010101000000-000000000000
	github.com/andrew-craig/cairn/services/users v0.0.0-00010101000000-000000000000
	github.com/google/uuid v1.6.0
	github.com/lib/pq v1.10.9
	github.com/mmcdole/gofeed v1.3.0
)

require (
	github.com/PuerkitoBio/goquery v1.9.1 // indirect
	github.com/andybalholm/cascadia v1.3.2 // indirect
	github.com/cenkalti/backoff/v4 v4.3.0 // indirect
	github.com/go-jose/go-jose/v4 v4.1.1 // indirect
	github.com/golang-jwt/jwt/v5 v5.3.0 // indirect
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
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/mmcdole/goxpp v1.1.1 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/ryanuber/go-glob v1.0.0 // indirect
	golang.org/x/crypto v0.46.0 // indirect
	golang.org/x/net v0.47.0 // indirect
	golang.org/x/text v0.32.0 // indirect
	golang.org/x/time v0.12.0 // indirect
)

// Use local version of user service for auth package
replace github.com/andrew-craig/cairn/services/users => ../users

// Use local version of shared cairn packages
replace github.com/andrew-craig/cairn => ../..
