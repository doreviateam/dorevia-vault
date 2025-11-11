#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

echo "🔄 Pulling latest changes..."
git pull

echo "🔨 Building binary with metadata..."
# Utiliser le script build.sh pour injecter les métadonnées
./scripts/build.sh

echo "🚀 Restarting service..."
sudo systemctl restart dorevia-vault

echo "✅ Deployment complete!"

