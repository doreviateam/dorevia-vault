#!/bin/bash

# Script de configuration du service systemd Dorevia Vault
# Usage: sudo ./scripts/configure_service.sh

set -e

SERVICE_FILE="/etc/systemd/system/dorevia-vault.service"
BACKUP_FILE="/etc/systemd/system/dorevia-vault.service.backup"

# Couleurs
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "🔧 Configuration du service Dorevia Vault"
echo "=========================================="
echo ""

# Vérifier que le script est exécuté en root
if [ "$EUID" -ne 0 ]; then 
    echo -e "${RED}❌ Erreur: Ce script doit être exécuté avec sudo${NC}"
    echo "Usage: sudo ./scripts/configure_service.sh"
    exit 1
fi

# Vérifier que le fichier service existe
if [ ! -f "$SERVICE_FILE" ]; then
    echo -e "${RED}❌ Erreur: Fichier service non trouvé: $SERVICE_FILE${NC}"
    exit 1
fi

# Créer une sauvegarde
echo "📋 Création d'une sauvegarde..."
cp "$SERVICE_FILE" "$BACKUP_FILE"
echo -e "${GREEN}✅ Sauvegarde créée: $BACKUP_FILE${NC}"
echo ""

# Vérifier la configuration actuelle
echo "📊 Configuration actuelle:"
if grep -q "DATABASE_URL" "$SERVICE_FILE"; then
    echo -e "${YELLOW}⚠️  DATABASE_URL déjà configuré${NC}"
    grep "DATABASE_URL" "$SERVICE_FILE"
else
    echo -e "${YELLOW}⚠️  DATABASE_URL non configuré${NC}"
fi

if grep -q "AUTH_ENABLED" "$SERVICE_FILE"; then
    echo -e "${YELLOW}⚠️  AUTH_ENABLED déjà configuré${NC}"
    grep "AUTH_ENABLED" "$SERVICE_FILE"
else
    echo -e "${YELLOW}⚠️  AUTH_ENABLED non configuré${NC}"
fi
echo ""

# Configuration DATABASE_URL
if ! grep -q "DATABASE_URL" "$SERVICE_FILE"; then
    echo "🔐 Configuration DATABASE_URL"
    echo "----------------------------"
    read -p "Entrez votre DATABASE_URL PostgreSQL (ou appuyez sur Entrée pour ignorer): " db_url
    
    if [ -n "$db_url" ]; then
        # Ajouter DATABASE_URL après Environment=PORT=8080
        sed -i "/Environment=PORT=8080/a Environment=\"DATABASE_URL=$db_url\"" "$SERVICE_FILE"
        echo -e "${GREEN}✅ DATABASE_URL ajouté${NC}"
    else
        echo -e "${YELLOW}⚠️  DATABASE_URL ignoré (les endpoints DB ne seront pas disponibles)${NC}"
    fi
    echo ""
fi

# Configuration AUTH_ENABLED
if ! grep -q "AUTH_ENABLED" "$SERVICE_FILE"; then
    echo "🔒 Configuration Authentification"
    echo "--------------------------------"
    echo "Souhaitez-vous activer l'authentification ?"
    echo "  - false : Accès libre aux endpoints (développement)"
    echo "  - true  : Authentification JWT/API Key requise (production)"
    read -p "AUTH_ENABLED [false]: " auth_enabled
    
    auth_enabled=${auth_enabled:-false}
    
    if [ "$auth_enabled" = "true" ]; then
        # Ajouter AUTH_ENABLED
        if grep -q "DATABASE_URL" "$SERVICE_FILE"; then
            sed -i "/DATABASE_URL/a Environment=\"AUTH_ENABLED=true\"" "$SERVICE_FILE"
        else
            sed -i "/Environment=PORT=8080/a Environment=\"AUTH_ENABLED=true\"" "$SERVICE_FILE"
        fi
        echo -e "${GREEN}✅ AUTH_ENABLED=true ajouté${NC}"
        echo ""
        echo "⚠️  Note: Si AUTH_ENABLED=true, vous devrez aussi configurer:"
        echo "   - AUTH_JWT_PUBLIC_KEY_PATH (pour JWT)"
        echo "   - Ou utiliser des API Keys"
    else
        # Ajouter AUTH_ENABLED=false
        if grep -q "DATABASE_URL" "$SERVICE_FILE"; then
            sed -i "/DATABASE_URL/a Environment=\"AUTH_ENABLED=false\"" "$SERVICE_FILE"
        else
            sed -i "/Environment=PORT=8080/a Environment=\"AUTH_ENABLED=false\"" "$SERVICE_FILE"
        fi
        echo -e "${GREEN}✅ AUTH_ENABLED=false ajouté${NC}"
    fi
    echo ""
fi

# Afficher la configuration finale
echo "📋 Configuration finale:"
echo "----------------------"
grep -E "Environment=" "$SERVICE_FILE" || echo "Aucune variable d'environnement"
echo ""

# Demander confirmation pour recharger et redémarrer
echo "🔄 Recharger et redémarrer le service ?"
read -p "Continuer [O/n]: " confirm

if [[ "$confirm" =~ ^[Nn]$ ]]; then
    echo -e "${YELLOW}⚠️  Service non redémarré. Redémarrez manuellement avec:${NC}"
    echo "   sudo systemctl daemon-reload"
    echo "   sudo systemctl restart dorevia-vault"
    exit 0
fi

# Recharger systemd
echo "🔄 Rechargement de systemd..."
systemctl daemon-reload
echo -e "${GREEN}✅ systemd rechargé${NC}"

# Redémarrer le service
echo "🔄 Redémarrage du service..."
systemctl restart dorevia-vault
echo -e "${GREEN}✅ Service redémarré${NC}"

# Vérifier le statut
echo ""
echo "📊 Statut du service:"
systemctl status dorevia-vault --no-pager | head -10

echo ""
echo -e "${GREEN}✅ Configuration terminée !${NC}"
echo ""
echo "🧪 Tests recommandés:"
echo "   curl https://vault.doreviateam.com/health"
if grep -q "DATABASE_URL" "$SERVICE_FILE"; then
    echo "   curl https://vault.doreviateam.com/dbhealth"
    echo "   curl https://vault.doreviateam.com/documents"
fi
echo ""

