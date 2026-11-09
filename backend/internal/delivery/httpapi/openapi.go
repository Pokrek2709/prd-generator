package httpapi

import "embed"

//go:embed openapi.yaml
var openAPIFiles embed.FS

func openAPISpec() ([]byte, error) { return openAPIFiles.ReadFile("openapi.yaml") }
