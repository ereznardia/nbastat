FROM golang:1.24.2-alpine

WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod ./
COPY go.sum ./

# Download dependencies
RUN go mod download

# Copy the rest of your app
COPY . ./

# Build your app
RUN go build -o app .

# Run your app
CMD ["./app"]
