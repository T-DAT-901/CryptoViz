# Multi-stage build pour optimiser la taille de l'image
FROM golang:1.21-alpine AS builder

# Installer les dépendances système
RUN apk add --no-cache git ca-certificates tzdata

# Définir le répertoire de travail
WORKDIR /app

# Copier les fichiers de dépendances et le code source
COPY go.mod go.sum* ./
COPY . .

# Télécharger et nettoyer les dépendances
RUN go mod tidy

# Compiler l'application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/server

# Image finale légère
FROM alpine:latest

# Installer ca-certificates pour les connexions HTTPS
RUN apk --no-cache add ca-certificates tzdata

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
