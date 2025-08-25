# Multi-stage build pour Vue.js
FROM node:18-alpine AS builder

# Définir le répertoire de travail
WORKDIR /app

# Copier les fichiers de dépendances
COPY package*.json ./

# Installer toutes les dépendances (dev + prod pour le build)
RUN npm install

# Copier le code source
COPY . .

# Build de l'application Vue.js
RUN npm run build

# Nettoyer les node_modules après le build
RUN rm -rf node_modules

# Image finale avec Nginx
FROM nginx:alpine

# Copier la configuration Nginx personnalisée
COPY nginx.conf /etc/nginx/nginx.conf

# Copier les fichiers buildés depuis l'étape précédente
COPY --from=builder /app/dist /usr/share/nginx/html

# Exposer le port
EXPOSE 80

# Commande par défaut (Nginx démarre automatiquement)
CMD ["nginx", "-g", "daemon off;"]
