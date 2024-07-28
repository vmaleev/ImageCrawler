# Build stage
FROM golang:1.21-alpine3.20 AS builder
WORKDIR /app

# Install dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code and build the binary
COPY . .
RUN go build -o main .

# Final stage
FROM alpine:latest
WORKDIR /root/s

# Copy the built binary
COPY --from=builder /app/main .
EXPOSE 8080

# Command to run the binary
CMD ["./main"]