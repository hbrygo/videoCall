# Utiliser l'image officielle Go
FROM golang:1.23-alpine

# Définir le répertoire de travail
WORKDIR /app

# Copier les fichiers sources
COPY . .

# Installer les dépendances
RUN go get github.com/gorilla/websocket && \
    go get github.com/pion/webrtc/v4 && \
    go mod tidy

# # Build l'application
# RUN go build -o main .

# Exposer le port
EXPOSE 8080

# Commande de démarrage
CMD ["go", "run", "main.go"]