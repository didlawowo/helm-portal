#!/bin/bash
set -e

# Script pour copier le fichier config.yaml dans le dossier helm avant l'empaquetage
# et s'assurer que la structure de la ConfigMap est correcte

echo "🔍 Vérification des répertoires..."
mkdir -p helm

echo "📝 Copie du fichier de configuration..."
cp src/config/config.yaml helm/

echo "✅ Fichier de configuration copié avec succès!"
echo "📦 Vous pouvez maintenant empaqueter le chart Helm avec: helm package ./helm"
