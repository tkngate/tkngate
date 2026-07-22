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
# Stage 2: Minimal production image — optimised for K8s sidecar deployment
FROM alpine:3.21

LABEL org.opencontainers.image.source="https://github.com/tkngate/tkngate"
LABEL org.opencontainers.image.title="TknGate"
LABEL org.opencontainers.image.description="Zero-Trust Kubernetes Sidecar for Enterprise AI Agent Credentials"
LABEL org.opencontainers.image.version="2.8.1"
LABEL io.tkngate.deployment-mode="sidecar"

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app
COPY --from=builder /build/tkngate /app/tkngate
COPY tkngate.example.yaml /app/tkngate.yaml

EXPOSE 7477 7478

HEALTHCHECK --interval=15s --timeout=3s --start-period=5s --retries=3 \
  CMD wget -qO- http://localhost:7478/healthz || exit 1

ENTRYPOINT ["/app/tkngate"]
CMD ["serve"]
