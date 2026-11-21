// Package tools contains code generation directives for the project.
// Run `go generate ./...` from the repository root to regenerate all files.
package generate

//go:generate go run ./cmd/reset/main.go -i ".git .github api migrations pkg profiles"
