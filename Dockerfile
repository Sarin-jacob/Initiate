# ==========================================
# STAGE 1: Build the Svelte/Vite Frontend
# ==========================================
FROM node:20-alpine AS frontend-builder
WORKDIR /build/frontend

# Copy package files and install dependencies
COPY frontend/package*.json ./
RUN npm ci

# Copy the rest of the frontend source code
COPY frontend/ ./

# Build the frontend. 
# Vite's outDir is '../static', so this creates /build/static
RUN npm run build

# ==========================================
# STAGE 2: Build the Go Backend
# ==========================================
FROM golang:1.26-alpine AS backend-builder

# Install GCC for SQLite/CGO support
RUN apk add --no-cache gcc musl-dev

WORKDIR /build/backend/server

# Copy go mod files to cache dependency downloads
COPY backend/server/go.mod backend/server/go.sum ./
RUN go mod download

# Copy the backend source code
COPY backend/server/ ./

# Build the binary from the cmd folder
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-w -s" -o /build/nexus-server ./cmd/main.go

# ==========================================
# STAGE 3: Final Minimal Runtime
# ==========================================
FROM alpine:latest

# Install CA certificates for external API calls and mail servers
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Create the persistent data directory for SQLite
RUN mkdir -p /app/data

# Copy the compiled Go binary
COPY --from=backend-builder /build/nexus-server .

# Copy the compiled frontend directly into the ./static folder where Go expects it
COPY --from=frontend-builder /build/static ./static

EXPOSE 8080

CMD ["./nexus-server"]