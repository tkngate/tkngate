# ============================================
# TKNGATE — Multi-Stage Production Dockerfile
# ============================================
# Stage 1: Build the Go binary
FROM golang:1.26-alpine AS builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-s -w" -o tkngate .

# ============================================
# Stage 2: Minimal production image
FROM alpine:3.21

LABEL org.opencontainers.image.source="https://github.com/tkngate/tkngate"

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app
COPY --from=builder /build/tkngate /app/tkngate
COPY tkngate.example.yaml /app/tkngate.yaml

EXPOSE 7477 8081

ENTRYPOINT ["/app/tkngate"]
CMD ["serve"]
