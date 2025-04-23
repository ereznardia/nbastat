# === Stage 1: Build frontend ===
FROM node:20-alpine as frontend-builder

WORKDIR /app
COPY frontend/ .
RUN npm install && npm run build

# === Stage 2: Build Go backend and package everything ===
FROM golang:1.24-alpine

WORKDIR /app

# Copy Vue build output into Go app's static directory
COPY --from=frontend-builder /app/dist ./frontend/dist

# Copy Go source code
COPY go.mod go.sum ./
RUN go mod download
COPY . .

# Build Go app
RUN go build -o server .

# Expose port (optional)
EXPOSE 8080

# Run the server
CMD ["./server"]
