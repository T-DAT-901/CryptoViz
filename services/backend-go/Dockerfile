# Multi-stage build pour optimiser la taille de l'image
FROM golang:1.23-alpine AS builder

# Installer les dépendances système (including librdkafka for confluent-kafka-go)
RUN apk add --no-cache git ca-certificates tzdata build-base librdkafka-dev pkgconf

# Définir le répertoire de travail
WORKDIR /app

# Copier les fichiers de dépendances et le code source
COPY go.mod go.sum* ./
COPY . .

# Télécharger et nettoyer les dépendances
RUN go mod tidy

# Compiler l'application (CGO_ENABLED=1 required for confluent-kafka-go)
# Use -tags dynamic to link against system's librdkafka (musl-compatible)
RUN CGO_ENABLED=1 GOOS=linux go build -tags dynamic -a -o main ./cmd/server

# Image finale légère
FROM alpine:latest

# Installer ca-certificates et librdkafka pour confluent-kafka-go
RUN apk --no-cache add ca-certificates tzdata librdkafka

# Créer un utilisateur non-root
RUN adduser -D -s /bin/sh appuser

# Définir le répertoire de travail
WORKDIR /root/

# Copier l'exécutable depuis l'étape de build
COPY --from=builder /app/main .

# Changer vers l'utilisateur non-root
USER appuser

# Exposer le port
EXPOSE 8080

# Commande par défaut
CMD ["./main"]
