#!/bin/bash

# Script de build Dorevia Vault avec injection de métadonnées via ldflags
# Usage: ./scripts/build.sh [version] [output]
#   version: Version à injecter (défaut: détecté depuis git tag ou 1.3.0)
#   output:  Fichier de sortie (défaut: bin/vault)

set -e

# Couleurs
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$PROJECT_ROOT"

# Paramètres
VERSION="${1:-}"
OUTPUT="${2:-bin/vault}"

# Détection automatique de la version
if [ -z "$VERSION" ]; then
    # Essayer de récupérer depuis git tag
    if git describe --tags --exact-match HEAD 2>/dev/null; then
        VERSION=$(git describe --tags --exact-match HEAD | sed 's/^v//')
    elif git describe --tags 2>/dev/null; then
        VERSION=$(git describe --tags | sed 's/^v//' | sed 's/-.*//')
    else
        # Version par défaut
        VERSION="1.3.0"
    fi
fi

# Récupération du commit hash
COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "dev")

# Date de build (UTC)
BUILT_AT=$(date -u +%Y-%m-%dT%H:%M:%SZ)

# Schema (format: YYYYMMDD_HHMM)
SCHEMA=$(date -u +%Y%m%d)_$(date -u +%H%M)

# Package path pour ldflags
PACKAGE_PATH="github.com/doreviateam/dorevia-vault/internal/buildinfo"

echo "🔨 Build Dorevia Vault"
echo "======================"
echo ""
echo "📦 Version    : $VERSION"
echo "🔖 Commit     : $COMMIT"
echo "📅 Built At   : $BUILT_AT"
echo "📋 Schema     : $SCHEMA"
echo "📁 Output     : $OUTPUT"
echo ""

# Construction des ldflags
LDFLAGS="-X $PACKAGE_PATH.Version=$VERSION"
LDFLAGS="$LDFLAGS -X $PACKAGE_PATH.Commit=$COMMIT"
LDFLAGS="$LDFLAGS -X $PACKAGE_PATH.BuiltAt=$BUILT_AT"
LDFLAGS="$LDFLAGS -X $PACKAGE_PATH.Schema=$SCHEMA"

# Build
echo "🔨 Compilation en cours..."
if go build -ldflags "$LDFLAGS" -o "$OUTPUT" ./cmd/vault; then
    echo ""
    echo -e "${GREEN}✅ Build réussi !${NC}"
    echo ""
    
    # Afficher les informations du binaire
    if [ -f "$OUTPUT" ]; then
        SIZE=$(ls -lh "$OUTPUT" | awk '{print $5}')
        echo "📊 Informations du binaire :"
        echo "   Taille : $SIZE"
        echo "   Chemin : $OUTPUT"
        echo ""
        
        # Vérifier que les valeurs sont bien injectées (optionnel)
        echo "🧪 Test de l'endpoint /version (si le service tourne) :"
        echo "   curl http://localhost:8080/version"
        echo ""
    fi
    
    # Afficher les valeurs injectées
    echo "📋 Valeurs injectées :"
    echo "   Version: $VERSION"
    echo "   Commit:  $COMMIT"
    echo "   BuiltAt:  $BUILT_AT"
    echo "   Schema:   $SCHEMA"
    echo ""
    
    exit 0
else
    echo ""
    echo -e "${YELLOW}❌ Erreur lors du build${NC}"
    exit 1
fi

