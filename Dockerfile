FROM golang:1.23-alpine AS builder

# Installation des dépendances système nécessaires
RUN apk add --no-cache git

WORKDIR /app

# Copier d'abord les fichiers de dépendances
COPY src/go.mod src/go.sum ./

# Télécharger les dépendances
RUN go mod download

# Installer orchestrion
RUN go install github.com/DataDog/orchestrion@latest

# Copier le code source
COPY src/ .

# Exécuter orchestrion pin
RUN orchestrion pin

# Construction
RUN CGO_ENABLED=0 GOOS=linux go build -o helm-portal ./cmd/server/main.go
# RUN orchestrion go build -o helm-portal ./cmd/server/main.go

# Image finale
FROM alpine:latest AS production

# Créer un utilisateur non-root
RUN adduser -D app

WORKDIR /app

# Copier l'exécutable depuis le builder
COPY --from=builder /app/helm-portal .

# Définir les permissions
RUN chown -R app:app /app

# Utiliser l'utilisateur non-root
USER app

EXPOSE 3030

CMD ["./helm-portal"]
