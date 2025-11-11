#!/bin/bash
# Script de configuration des variables d'environnement pour Dorevia Vault
# Usage: source setup_env.sh
# Date: Janvier 2025

set -euo pipefail

# Couleurs pour les messages
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "🔧 Configuration des variables d'environnement Dorevia Vault"
echo "============================================================"
echo ""

# ============================================================================
# CONFIGURATION DE BASE
# ============================================================================

export PORT="${PORT:-8080}"
export LOG_LEVEL="${LOG_LEVEL:-info}"
export STORAGE_DIR="${STORAGE_DIR:-/opt/dorevia-vault/storage}"

echo "✅ Configuration de base:"
echo "   PORT=$PORT"
echo "   LOG_LEVEL=$LOG_LEVEL"
echo "   STORAGE_DIR=$STORAGE_DIR"
echo ""

# ============================================================================
# CONFIGURATION POSTGRESQL
# ============================================================================

if [ -z "${DATABASE_URL:-}" ]; then
    echo -e "${YELLOW}⚠️  DATABASE_URL non configuré${NC}"
    echo "   Format attendu: postgres://user:password@host:port/database?sslmode=disable"
    echo ""
    read -p "Entrez votre DATABASE_URL (ou appuyez sur Entrée pour ignorer): " db_url
    
    if [ -n "$db_url" ]; then
        export DATABASE_URL="$db_url"
        echo -e "${GREEN}✅ DATABASE_URL configuré${NC}"
    else
        echo -e "${RED}❌ DATABASE_URL non configuré - certaines fonctionnalités ne seront pas disponibles${NC}"
    fi
else
    echo -e "${GREEN}✅ DATABASE_URL déjà configuré${NC}"
    # Masquer le mot de passe dans l'affichage
    echo "   $(echo $DATABASE_URL | sed 's/:[^:@]*@/:***@/g')"
fi
echo ""

# ============================================================================
# CONFIGURATION JWS
# ============================================================================

export JWS_ENABLED="${JWS_ENABLED:-true}"
export JWS_REQUIRED="${JWS_REQUIRED:-true}"
export JWS_KID="${JWS_KID:-key-2025-Q1}"

# Chemins des clés RSA
PRIVATE_KEY_PATH="/opt/dorevia-vault/keys/private.pem"
PUBLIC_KEY_PATH="/opt/dorevia-vault/keys/public.pem"

# Vérifier si les clés existent
if [ -f "$PRIVATE_KEY_PATH" ] && [ -f "$PUBLIC_KEY_PATH" ]; then
    export JWS_PRIVATE_KEY_PATH="${JWS_PRIVATE_KEY_PATH:-$PRIVATE_KEY_PATH}"
    export JWS_PUBLIC_KEY_PATH="${JWS_PUBLIC_KEY_PATH:-$PUBLIC_KEY_PATH}"
    echo -e "${GREEN}✅ Clés RSA trouvées${NC}"
    echo "   JWS_PRIVATE_KEY_PATH=$JWS_PRIVATE_KEY_PATH"
    echo "   JWS_PUBLIC_KEY_PATH=$JWS_PUBLIC_KEY_PATH"
    echo "   JWS_KID=$JWS_KID"
else
    echo -e "${RED}❌ Clés RSA non trouvées dans $PRIVATE_KEY_PATH${NC}"
    echo "   Pour générer les clés:"
    echo "   go run ./cmd/keygen/main.go --out /opt/dorevia-vault/keys --kid key-2025-Q1 --bits 2048"
    export JWS_ENABLED="false"
    export JWS_REQUIRED="false"
fi
echo ""

# ============================================================================
# CONFIGURATION LEDGER
# ============================================================================

export LEDGER_ENABLED="${LEDGER_ENABLED:-true}"

echo "✅ Configuration Ledger:"
echo "   LEDGER_ENABLED=$LEDGER_ENABLED"
echo ""

# ============================================================================
# CONFIGURATION SPRINT 5 - AUTHENTIFICATION & AUTORISATION
# ============================================================================

export AUTH_ENABLED="${AUTH_ENABLED:-false}"
export AUTH_JWT_ENABLED="${AUTH_JWT_ENABLED:-true}"
export AUTH_APIKEY_ENABLED="${AUTH_APIKEY_ENABLED:-true}"

# Chemin clé publique JWT (peut être la même que JWS)
JWT_PUBLIC_KEY_PATH="${AUTH_JWT_PUBLIC_KEY_PATH:-${JWS_PUBLIC_KEY_PATH:-/opt/dorevia-vault/keys/public.pem}}"
if [ -f "$JWT_PUBLIC_KEY_PATH" ]; then
    export AUTH_JWT_PUBLIC_KEY_PATH="$JWT_PUBLIC_KEY_PATH"
    echo -e "${GREEN}✅ Clé publique JWT trouvée${NC}"
    echo "   AUTH_JWT_PUBLIC_KEY_PATH=$AUTH_JWT_PUBLIC_KEY_PATH"
else
    echo -e "${YELLOW}⚠️  Clé publique JWT non trouvée${NC}"
    echo "   AUTH_JWT_PUBLIC_KEY_PATH non configuré"
fi

echo "✅ Configuration Authentification:"
echo "   AUTH_ENABLED=$AUTH_ENABLED"
echo "   AUTH_JWT_ENABLED=$AUTH_JWT_ENABLED"
echo "   AUTH_APIKEY_ENABLED=$AUTH_APIKEY_ENABLED"
echo ""

# ============================================================================
# CONFIGURATION SPRINT 5 - HASHICORP VAULT (Optionnel)
# ============================================================================

export VAULT_ENABLED="${VAULT_ENABLED:-false}"
export VAULT_ADDR="${VAULT_ADDR:-}"
export VAULT_TOKEN="${VAULT_TOKEN:-}"
export VAULT_KEY_PATH="${VAULT_KEY_PATH:-secret/data/dorevia/keys}"

if [ "$VAULT_ENABLED" = "true" ]; then
    if [ -z "$VAULT_ADDR" ] || [ -z "$VAULT_TOKEN" ]; then
        echo -e "${YELLOW}⚠️  Vault activé mais VAULT_ADDR ou VAULT_TOKEN non configuré${NC}"
        echo "   Format VAULT_ADDR: https://vault.example.com:8200"
        echo "   Format VAULT_TOKEN: hvs.xxxxx"
    else
        echo -e "${GREEN}✅ Configuration Vault détectée${NC}"
        echo "   VAULT_ADDR=$VAULT_ADDR"
        echo "   VAULT_KEY_PATH=$VAULT_KEY_PATH"
    fi
else
    echo "✅ Vault: Désactivé (utilisation fichiers locaux)"
fi
echo ""

# ============================================================================
# CONFIGURATION SPRINT 5 - VALIDATION FACTUR-X
# ============================================================================

export FACTURX_VALIDATION_ENABLED="${FACTURX_VALIDATION_ENABLED:-true}"
export FACTURX_VALIDATION_REQUIRED="${FACTURX_VALIDATION_REQUIRED:-false}"

echo "✅ Configuration Factur-X:"
echo "   FACTURX_VALIDATION_ENABLED=$FACTURX_VALIDATION_ENABLED"
echo "   FACTURX_VALIDATION_REQUIRED=$FACTURX_VALIDATION_REQUIRED"
echo ""

# ============================================================================
# CONFIGURATION SPRINT 5 - WEBHOOKS ASYNCHRONES
# ============================================================================

export WEBHOOKS_ENABLED="${WEBHOOKS_ENABLED:-false}"
export WEBHOOKS_REDIS_URL="${WEBHOOKS_REDIS_URL:-redis://localhost:6379/0}"
export WEBHOOKS_SECRET_KEY="${WEBHOOKS_SECRET_KEY:-}"
export WEBHOOKS_WORKERS="${WEBHOOKS_WORKERS:-3}"
export WEBHOOKS_URLS="${WEBHOOKS_URLS:-}"

if [ "$WEBHOOKS_ENABLED" = "true" ]; then
    if [ -z "$WEBHOOKS_SECRET_KEY" ]; then
        echo -e "${YELLOW}⚠️  Webhooks activés mais WEBHOOKS_SECRET_KEY non configuré${NC}"
        echo "   Pour générer une clé: openssl rand -hex 32"
    else
        echo -e "${GREEN}✅ Configuration Webhooks détectée${NC}"
        echo "   WEBHOOKS_REDIS_URL=$WEBHOOKS_REDIS_URL"
        echo "   WEBHOOKS_WORKERS=$WEBHOOKS_WORKERS"
        if [ -n "$WEBHOOKS_URLS" ]; then
            echo "   WEBHOOKS_URLS configuré"
        else
            echo -e "${YELLOW}   WEBHOOKS_URLS non configuré${NC}"
        fi
    fi
else
    echo "✅ Webhooks: Désactivés"
fi
echo ""

# ============================================================================
# VÉRIFICATIONS
# ============================================================================

echo "🔍 Vérifications..."
echo ""

# Vérifier répertoire storage
if [ -d "$STORAGE_DIR" ]; then
    echo -e "${GREEN}✅ Répertoire storage existe: $STORAGE_DIR${NC}"
else
    echo -e "${YELLOW}⚠️  Répertoire storage inexistant: $STORAGE_DIR${NC}"
    read -p "Créer le répertoire maintenant? (o/N): " create_storage
    if [[ "$create_storage" =~ ^[Oo]$ ]]; then
        mkdir -p "$STORAGE_DIR"
        echo -e "${GREEN}✅ Répertoire créé${NC}"
    fi
fi
echo ""

# Vérifier PostgreSQL si DATABASE_URL configuré
if [ -n "${DATABASE_URL:-}" ]; then
    echo "🔍 Test de connexion PostgreSQL..."
    if command -v psql &> /dev/null; then
        if psql "$DATABASE_URL" -c "SELECT version();" &> /dev/null; then
            echo -e "${GREEN}✅ Connexion PostgreSQL OK${NC}"
            
            # Vérifier les tables
            if psql "$DATABASE_URL" -c "\dt" 2>&1 | grep -q "documents"; then
                echo -e "${GREEN}✅ Table 'documents' présente${NC}"
            else
                echo -e "${YELLOW}⚠️  Table 'documents' absente - migrations à appliquer${NC}"
            fi
            
            if psql "$DATABASE_URL" -c "\dt" 2>&1 | grep -q "ledger"; then
                echo -e "${GREEN}✅ Table 'ledger' présente${NC}"
            else
                echo -e "${YELLOW}⚠️  Table 'ledger' absente - migrations à appliquer${NC}"
            fi
        else
            echo -e "${RED}❌ Connexion PostgreSQL échouée${NC}"
            echo "   Vérifiez votre DATABASE_URL"
        fi
    else
        echo -e "${YELLOW}⚠️  psql non disponible - impossible de tester PostgreSQL${NC}"
    fi
    echo ""
fi

# ============================================================================
# RÉSUMÉ
# ============================================================================

echo "============================================================"
echo "📋 Résumé de la configuration"
echo "============================================================"
echo ""

# Variables de base
echo "Configuration de base:"
echo "  PORT=$PORT"
echo "  LOG_LEVEL=$LOG_LEVEL"
echo "  STORAGE_DIR=$STORAGE_DIR"
echo ""

# PostgreSQL
if [ -n "${DATABASE_URL:-}" ]; then
    echo -e "${GREEN}✅ PostgreSQL: Configuré${NC}"
    echo "  DATABASE_URL=$(echo $DATABASE_URL | sed 's/:[^:@]*@/:***@/g')"
else
    echo -e "${RED}❌ PostgreSQL: Non configuré${NC}"
fi
echo ""

# JWS
if [ "$JWS_ENABLED" = "true" ] && [ -f "${JWS_PRIVATE_KEY_PATH:-}" ]; then
    echo -e "${GREEN}✅ JWS: Activé${NC}"
    echo "  JWS_PRIVATE_KEY_PATH=$JWS_PRIVATE_KEY_PATH"
    echo "  JWS_PUBLIC_KEY_PATH=$JWS_PUBLIC_KEY_PATH"
    echo "  JWS_KID=$JWS_KID"
else
    echo -e "${YELLOW}⚠️  JWS: Désactivé${NC}"
fi
echo ""

# Ledger
echo "Ledger:"
echo "  LEDGER_ENABLED=$LEDGER_ENABLED"
echo ""

# Sprint 5 - Authentification
if [ "$AUTH_ENABLED" = "true" ]; then
    echo -e "${GREEN}✅ Authentification: Activée${NC}"
    echo "  AUTH_ENABLED=$AUTH_ENABLED"
    echo "  AUTH_JWT_ENABLED=$AUTH_JWT_ENABLED"
    echo "  AUTH_APIKEY_ENABLED=$AUTH_APIKEY_ENABLED"
    if [ -n "${AUTH_JWT_PUBLIC_KEY_PATH:-}" ]; then
        echo "  AUTH_JWT_PUBLIC_KEY_PATH=$AUTH_JWT_PUBLIC_KEY_PATH"
    fi
else
    echo -e "${YELLOW}⚠️  Authentification: Désactivée${NC}"
fi
echo ""

# Sprint 5 - Vault
if [ "$VAULT_ENABLED" = "true" ]; then
    echo -e "${GREEN}✅ Vault: Activé${NC}"
    echo "  VAULT_ADDR=$VAULT_ADDR"
    echo "  VAULT_KEY_PATH=$VAULT_KEY_PATH"
else
    echo "Vault: Désactivé (fichiers locaux)"
fi
echo ""

# Sprint 5 - Factur-X
echo "Factur-X:"
echo "  FACTURX_VALIDATION_ENABLED=$FACTURX_VALIDATION_ENABLED"
echo "  FACTURX_VALIDATION_REQUIRED=$FACTURX_VALIDATION_REQUIRED"
echo ""

# Sprint 5 - Webhooks
if [ "$WEBHOOKS_ENABLED" = "true" ]; then
    echo -e "${GREEN}✅ Webhooks: Activés${NC}"
    echo "  WEBHOOKS_REDIS_URL=$WEBHOOKS_REDIS_URL"
    echo "  WEBHOOKS_WORKERS=$WEBHOOKS_WORKERS"
else
    echo "Webhooks: Désactivés"
fi
echo ""

# ============================================================================
# INSTRUCTIONS
# ============================================================================

echo "============================================================"
echo "📝 Instructions"
echo "============================================================"
echo ""
echo "Les variables d'environnement sont maintenant configurées dans cette session."
echo ""
echo "Pour les rendre permanentes, ajoutez ces lignes à votre ~/.bashrc ou ~/.profile:"
echo ""
echo "  export PORT=$PORT"
echo "  export LOG_LEVEL=$LOG_LEVEL"
echo "  export STORAGE_DIR=$STORAGE_DIR"
if [ -n "${DATABASE_URL:-}" ]; then
    echo "  export DATABASE_URL=\"$DATABASE_URL\""
fi
if [ "$JWS_ENABLED" = "true" ] && [ -n "${JWS_PRIVATE_KEY_PATH:-}" ]; then
    echo "  export JWS_ENABLED=$JWS_ENABLED"
    echo "  export JWS_REQUIRED=$JWS_REQUIRED"
    echo "  export JWS_PRIVATE_KEY_PATH=$JWS_PRIVATE_KEY_PATH"
    echo "  export JWS_PUBLIC_KEY_PATH=$JWS_PUBLIC_KEY_PATH"
    echo "  export JWS_KID=$JWS_KID"
fi
echo "  export LEDGER_ENABLED=$LEDGER_ENABLED"
if [ "$AUTH_ENABLED" = "true" ]; then
    echo "  export AUTH_ENABLED=$AUTH_ENABLED"
    echo "  export AUTH_JWT_ENABLED=$AUTH_JWT_ENABLED"
    echo "  export AUTH_APIKEY_ENABLED=$AUTH_APIKEY_ENABLED"
    if [ -n "${AUTH_JWT_PUBLIC_KEY_PATH:-}" ]; then
        echo "  export AUTH_JWT_PUBLIC_KEY_PATH=\"$AUTH_JWT_PUBLIC_KEY_PATH\""
    fi
fi
if [ "$VAULT_ENABLED" = "true" ]; then
    echo "  export VAULT_ENABLED=$VAULT_ENABLED"
    echo "  export VAULT_ADDR=\"$VAULT_ADDR\""
    echo "  export VAULT_TOKEN=\"$VAULT_TOKEN\""
    echo "  export VAULT_KEY_PATH=\"$VAULT_KEY_PATH\""
fi
echo "  export FACTURX_VALIDATION_ENABLED=$FACTURX_VALIDATION_ENABLED"
echo "  export FACTURX_VALIDATION_REQUIRED=$FACTURX_VALIDATION_REQUIRED"
if [ "$WEBHOOKS_ENABLED" = "true" ]; then
    echo "  export WEBHOOKS_ENABLED=$WEBHOOKS_ENABLED"
    echo "  export WEBHOOKS_REDIS_URL=\"$WEBHOOKS_REDIS_URL\""
    echo "  export WEBHOOKS_SECRET_KEY=\"$WEBHOOKS_SECRET_KEY\""
    echo "  export WEBHOOKS_WORKERS=$WEBHOOKS_WORKERS"
    if [ -n "$WEBHOOKS_URLS" ]; then
        echo "  export WEBHOOKS_URLS=\"$WEBHOOKS_URLS\""
    fi
fi
echo ""
echo "Ou utilisez ce script à chaque session:"
echo "  source /opt/dorevia-vault/setup_env.sh"
echo ""
echo "Pour tester le service:"
echo "  go run ./cmd/vault/main.go"
echo "  # ou"
echo "  ./bin/vault"
echo ""
echo -e "${GREEN}✅ Configuration terminée!${NC}"
echo ""

