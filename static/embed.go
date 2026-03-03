package static

import "embed"

// FS contiene el frontend (web/dist) embebido. Se rellena con `make build` (copia web/dist → static/dist).
//go:embed dist
var FS embed.FS
