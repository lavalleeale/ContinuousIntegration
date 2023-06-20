//go:build !noembed

package main

import (
	"embed"
)

//go:embed assets/output/*
var assetsFS embed.FS

//go:embed templates/*
var templatesFS embed.FS
