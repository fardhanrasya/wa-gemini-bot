# Build stage
FROM golang:1.25-alpine AS builder

# Install dependencies untuk CGO (diperlukan untuk sqlite3)
RUN apk add --no-cache gcc musl-dev sqlite-dev

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build binary dengan optimasi
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -ldflags="-w -s" -o wa-gemini-bot ./cmd/bot

# Runtime stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates sqlite-libs tzdata

WORKDIR /app

# Copy binary dari build stage
COPY --from=builder /app/wa-gemini-bot .

# Copy .env.example sebagai template (user harus mount .env sendiri)
COPY .env.example .

# Create directory untuk persistent data
RUN mkdir -p /app/data

# Create volume untuk persistent data
VOLUME ["/app/data"]

# Expose webhook port untuk DOKU (jika digunakan)
EXPOSE 8080

# Set timezone (optional, bisa di-override via env)
ENV TZ=Asia/Jakarta

# Run bot
CMD ["./wa-gemini-bot"]
