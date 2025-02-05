# Helm Portal

A lightweight web portal for managing and deploying Helm charts in your Kubernetes clusters.

## 🚀 Key Components

### Backend (Go + Fiber)

- **API Layer**
  - RESTful endpoints for CRUD operations on charts
  - Authentication middleware
  - Version management endpoints

- **Chart Management**
  - Chart parsing and validation
  - Version control system
  - Storage interface (file-based)

- **Kubernetes Integration**
  - Cluster connection management
  - Chart deployment handling
  - Status monitoring

### Frontend (HTML/CSS Templates)

- **Dashboard**
  - Available charts listing
  - Version history display
  - Chart metadata visualization

- **Interaction**
  - Chart upload interface
  - Search and filtering
  - Deployment configuration form

### Core Features

- 📦 Chart repository management
- 🔄 Version tracking and history
- 🔐 Basic authentication system
- 🔍 Search and filter capabilities
- 📊 Chart metadata display
- 🚀 Direct deployment to clusters

## Project Structure

```
helm-portal/
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── api/
│   │   ├── handlers/
│   │   ├── middleware/
│   │   └── routes/
│   ├── chart/
│   │   ├── parser/
│   │   └── storage/
│   ├── kubernetes/
│   │   └── client/
│   └── models/
├── web/
│   ├── templates/
│   └── static/
└── config/
    └── config.yaml
```

## Prerequisites

- Go 1.20+
- Kubernetes cluster access
- Helm 3.x

## Configuration

The application uses a YAML configuration file for settings such as:
- Server port and host
- Storage location for charts
- Authentication credentials
- Kubernetes cluster configurations

## Development Status

🚧 This project is currently under development. Contributions are welcome!

