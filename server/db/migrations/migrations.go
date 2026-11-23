// Package migrations embeds the numbered SQL migrations for the server.
// Migrations are append-only: never modify an existing file, add a new one.
package migrations

import "embed"

// FS holds every *.sql migration file in this directory.
//
//go:embed *.sql
var FS embed.FS
