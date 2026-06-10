# ── Stage 1: Build ────────────────────────────────────────────────────────────
FROM golang:1.21-alpine AS build
WORKDIR /app
COPY go.mod go.sum* ./
RUN go mod download 2>/dev/null || true
COPY *.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o process_scout .

# ── Stage 2: Runtime ──────────────────────────────────────────────────────────
FROM alpine:3.19
RUN apk add --no-cache ca-certificates procps
WORKDIR /app
RUN addgroup -S appgroup && adduser -S appuser -G appgroup
COPY --from=build /app/process_scout .
COPY config.yaml .
USER appuser
EXPOSE 9001
ENTRYPOINT ["./process_scout", "--config=config.yaml"]
