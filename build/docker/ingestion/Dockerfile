## MediaFlow Ingestion Service Dockerfile
## Builds a minimal container image using Go 1.25.1 and distroless runtime.

FROM golang:1.25.1-alpine AS builder

WORKDIR /src

RUN apk add --no-cache ca-certificates git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ENV CGO_ENABLED=0
RUN go build -trimpath -ldflags "-s -w" -o /out/ingestion ./cmd/ingestion

FROM gcr.io/distroless/base-debian12:nonroot
WORKDIR /app

COPY --from=builder /out/ingestion /app/ingestion

ENV APP_NAME=mediaflow-ingestion \
    APP_ENV=production \
    HTTP_ADDR=:8080 \
    GRPC_ADDR=:9090 \
    METRICS_ADDR=:9102

EXPOSE 8080 9090 9102

ENTRYPOINT ["/app/ingestion"]

