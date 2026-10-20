# go-sqlite3 requiere CGO habilitado y un compilador C -> golang:bookworm
# (trae gcc) en el stage de build. El binario resultante es estático y corre
# en una imagen final mínima (debian-slim con ca-certificates).
FROM golang:1.24-bookworm AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ENV CGO_ENABLED=1
RUN go build -o /out/api ./cmd/api

FROM debian:bookworm-slim

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /out/api ./api
COPY migrations ./migrations

EXPOSE 8080

ENTRYPOINT ["./api"]
