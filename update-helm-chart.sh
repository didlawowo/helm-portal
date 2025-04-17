#!/bin/bash
set -e

echo "🔄 Mise à jour du Helm Chart Helm Portal..."

# Copier le fichier config.yaml dans le dossier helm
echo "📝 Copie du fichier config.yaml..."
cp src/config/config.yaml helm/

# Variables
NAMESPACE="${1:-kube-infra}"
RELEASE_NAME="${2:-helm-portal}"

# Installation/Mise à jour du chart
echo "🛠️ Installation/Mise à jour du chart Helm..."
helm upgrade --install "$RELEASE_NAME" ./helm \
  --namespace "$NAMESPACE" \
  --create-namespace

echo "✅ Opération terminée avec succès!"
echo "🔍 Pour vérifier le déploiement: kubectl get pods -n $NAMESPACE"
