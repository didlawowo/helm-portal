FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY src/go.mod src/go.sum ./

# Télécharger les dépendances
RUN go mod download

# Installer orchestrion
RUN go install github.com/DataDog/orchestrion@latest

COPY src/ .

# Exécuter orchestrion pin pour datadog APM
RUN orchestrion pin

# RUN CGO_ENABLED=0 GOOS=linux go build -o helm-portal ./cmd/server/main.go
RUN orchestrion go build -o helm-portal ./cmd/server/main.go

# Image finale
FROM alpine:latest AS production

RUN adduser -D app -u 1000  -g app --home /app  && \
    apk add --no-cache ca-certificates && \
    rm -rf /var/cache/apk/*

WORKDIR /app
# Copier l'exécutable depuis le builder
COPY --from=builder /app/helm-portal .
COPY --from=builder /app/views ./views
RUN mkdir config
COPY --from=builder /app/config/config.yaml ./config/config.yaml

RUN chown -R app:app /app

USER app

EXPOSE 3030

CMD ["./helm-portal"]
