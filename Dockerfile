FROM golang:1.22-alpine AS builder

WORKDIR /app

# Install build deps
RUN apk add --no-cache git ca-certificates

# Copy go mod files first for layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -o xiantu-server ./cmd/server

# Production image
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/xiantu-server .
COPY --from=builder /app/public ./public

ENV TZ=Asia/Shanghai

EXPOSE 8080

CMD ["./xiantu-server"]
