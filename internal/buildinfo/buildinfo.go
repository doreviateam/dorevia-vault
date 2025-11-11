package buildinfo

// Ces valeurs sont surchargées au build via -ldflags.
var (
	Version = "0.0.1"  // -X github.com/doreviateam/dorevia-vault/internal/buildinfo.Version=1.3.0
	Commit  = "dev"    // -X github.com/doreviateam/dorevia-vault/internal/buildinfo.Commit=$(git rev-parse --short HEAD)
	BuiltAt = "unknown" // -X github.com/doreviateam/dorevia-vault/internal/buildinfo.BuiltAt=$(date -u +%Y-%m-%dT%H:%M:%SZ)
	Schema  = "unknown" // -X github.com/doreviateam/dorevia-vault/internal/buildinfo.Schema=2025-11-11_0012
)

// VersionPayload représente la réponse de l'endpoint /version
type VersionPayload struct {
	Version string `json:"version"`
	Commit  string `json:"commit"`
	BuiltAt string `json:"built_at"`
	Schema  string `json:"schema"`
}

