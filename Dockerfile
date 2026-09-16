FROM golang:1.23-alpine AS builder

WORKDIR /app

# Install system dependencies
RUN apk add --no-cache git

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/server/main.go

# Final stage
FROM alpine:latest

WORKDIR /app

# Install certificates and timezone data
RUN apk --no-cache add ca-certificates tzdata

# Copy the pre-built binary
COPY --from=builder /app/main .

# Expose port
EXPOSE 8000

# Command to run the application
CMD ["./main"]
