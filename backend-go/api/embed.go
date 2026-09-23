// Package api embeds the hand-maintained OpenAPI document of the public API.
package api

import _ "embed"

// OpenAPI is api/openapi.yaml (OpenAPI 3.1), served at /docs/openapi.yaml.
//
//go:embed openapi.yaml
var OpenAPI []byte
