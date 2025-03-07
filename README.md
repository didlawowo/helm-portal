 
# Helm Portal

Un registre OCI (Open Container Initiative) léger et autonome pour stocker, gérer et partager vos charts Helm.

## 📋 Description

Helm OCI Registry est une solution simple mais puissante qui vous permet d'héberger vos propres charts Helm dans un registre compatible OCI. Ce projet implémente les spécifications OCI pour permettre un stockage et une distribution efficaces des charts Helm sans dépendre de services externes.

## ✨ Fonctionnalités

- 📦 Registre OCI complet pour charts Helm
- 🔄 Gestion des versions et des tags
- 🔒 Authentification simple et sécurisée
- 🌐 API REST pour l'interaction programmatique
- 📊 Interface web pour la gestion et la visualisation des charts
- 🔍 Recherche et filtrage des charts disponibles
- sauvegarde sur bucket AWS / GCP

## 🛠️ Prérequis

- Kubernetes 1.18+
- Helm 3.8.0+ (support OCI)
- Docker (pour construire l'image si nécessaire)

## 🚀 Utilisation

### Avec Helm (recommandé)

```bash
# Ajouter le repo Helm  
helm repo add helm-oci-registry http://localhost:3030/

# Installer le chart
helm install helm-oci-registry helm-oci-registry/helm-oci-registry

# Ou directement depuis le répertoire local
helm install helm-oci-registry ./helm
```

### Avec le registre OCI

```bash
# Empaqueter le chart
helm package <votrechart>

# Se connecter au registre OCI
 helm registry login localhost:3031 \
  --username admin \
  --password admin123 \

# Pousser le chart vers le registre OCI
helm push ./votre-chart-1.1.0.tgz oci://localhost:3030
```

## 📝 Configuration

Le chart Helm accepte les valeurs de configuration suivantes:

```yaml
# values.yaml
service:
  type: ClusterIP
  port: 3030

persistence:
  enabled: true
  size: 10Gi

auth:
  enabled: true
  username: admin
  # Le mot de passe sera généré automatiquement si non spécifié
  # password: changeme

 
```

### Lister les charts disponibles

```bash
# Via l'API REST
curl -X GET http://localhost:3030/api/charts

# Ou utiliser l'interface web
# Naviguer vers http:/localhost:3030
```
 
## 🤝 Contribution

Les contributions sont les bienvenues! N'hésitez pas à ouvrir une issue ou une pull request.

## 📄 Licence

Ce projet est sous licence MIT.