# Helm Portal

Un registre OCI (Open Container Initiative) léger et autonome pour stocker, gérer et partager vos charts Helm.

## 📋 Description

Helm Portal est une solution simple mais puissante qui vous permet d'héberger vos propres charts Helm dans un registre compatible OCI. Ce projet implémente les spécifications OCI pour permettre un stockage et une distribution efficaces des charts Helm sans dépendre de services externes.

## ✨ Fonctionnalités

- 📦 Registre OCI complet pour charts Helm
- 🔄 Gestion des versions et des tags
- 🔒 Authentification simple et sécurisée
- 🌐 API REST pour l'interaction programmatique
- 📊 Interface web pour la gestion et la visualisation des charts
- 🔍 Recherche et filtrage des charts disponibles
- 💾 Sauvegarde sur bucket AWS / GCP
- 🔄 Backup simple avec un bouton dédié

## 🛠️ Prérequis

- Kubernetes 1.18+
- Helm 3.8.0+ (support OCI)
- Docker (pour construire l'image si nécessaire)

## 🚀 Installation

### Préparation du chart

Avant d'installer ou d'empaqueter le chart, exécutez notre script pour copier le fichier de configuration:

```bash
# Rendez le script exécutable
chmod +x scripts/copy-config.sh

# Exécutez le script
./scripts/copy-config.sh
```

### Installation avec notre script (recommandé)

```bash
# Rendez le script exécutable
chmod +x update-helm-chart.sh

# Installez ou mettez à jour le chart (avec namespace par défaut)
./update-helm-chart.sh

# Ou spécifiez un namespace et un nom de release
./update-helm-chart.sh mon-namespace mon-helm-portal
```

### Installation manuelle avec Helm

```bash
# Installer le chart
helm install helm-portal ./helm

# Ou avec un namespace spécifique
helm install helm-portal ./helm --namespace mon-namespace --create-namespace
```

### Utilisation du registre OCI

```bash
# Empaqueter votre chart
helm package <votrechart>

# Se connecter au registre OCI
helm registry login localhost:3030 \
  --username admin \
  --password admin123

# Pousser le chart vers le registre OCI
helm push ./votre-chart-1.0.0.tgz oci://localhost:3030
```

## 📝 Configuration

Le chart Helm utilise un fichier `config.yaml` pour sa configuration principale, qui est automatiquement intégré dans une ConfigMap lors de l'installation.

### Structure de la ConfigMap

Le fichier `src/config/config.yaml` est copié dans le chart Helm et utilisé comme base pour la ConfigMap. Les valeurs peuvent être remplacées par celles spécifiées dans `values.yaml`.

### Principales options de configuration

```yaml
# values.yaml
server:
  port: 3030

storage:
  path: "data"

auth:
  enabled: true
  users:
  - username: "admin"
    password: "admin123"

logging:
  level: "info"
  format: "text"

# Configuration optionnelle des sauvegardes
backup:
  gcp:
    enabled: false
    bucket: "helm-portal-backup"
    projectID: "votre-projet"
  # aws:
  #   bucket: "helm-portal-backup"
  #   region: "eu-west-1"
```

## 🧩 Utilisation

### Interface Web

L'interface web est accessible à l'adresse du service (par défaut `http://localhost:3030`) et permet:
- Visualiser tous les charts disponibles
- Télécharger des charts directement depuis l'interface
- Consulter les détails et les valeurs de chaque chart
- Effectuer des sauvegardes via le bouton dédié

### API REST

```bash
# Lister tous les charts
curl -X GET http://localhost:3030/api/charts

# Obtenir les détails d'un chart spécifique
curl -X GET http://localhost:3030/api/charts/nom-du-chart/version
```

### Commandes Helm

```bash
# Lister les charts disponibles dans le registre
helm search repo helm-portal

# Installer un chart depuis le registre
helm install mon-app oci://localhost:3030/nom-du-chart --version 1.0.0
```

## 🤝 Contribution

Les contributions sont les bienvenues! N'hésitez pas à ouvrir une issue ou une pull request.

## 📄 Licence

Ce projet est sous licence MIT.
