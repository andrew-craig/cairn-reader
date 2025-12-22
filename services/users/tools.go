//go:build tools
// +build tools

// This file ensures that development and migration tools are tracked in go.mod
package tools

import (
	_ "github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/golang/mock/gomock"
	_ "github.com/stretchr/testify"
)
