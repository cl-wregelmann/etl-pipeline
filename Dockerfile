# syntax=docker/dockerfile:1

FROM golang:1.26 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# modernc.org/sqlite is pure Go, so the binary is fully static and runs on distroless/alpine.
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/etl ./cmd/etl
# distroless has no shell to mkdir with; pre-create the output dir so prod can chown it.
RUN mkdir -p /out/data/out

# Local development only (never deploy): has a shell and the sqlite3 CLI so you can exec in and inspect data.
FROM alpine:3.22 AS dev
RUN apk add --no-cache sqlite
COPY --from=build /out/etl /usr/local/bin/etl
RUN mkdir -p /data/in /data/out
WORKDIR /data
ENTRYPOINT ["etl", "-source", "/data/in", "-db", "/data/out/etl.db"]

# Production image (default target): no shell, runs as non-root.
FROM gcr.io/distroless/static-debian12:nonroot AS prod
COPY --from=build /out/etl /etl
# Owned by nonroot so a fresh named volume inherits writable ownership.
COPY --from=build --chown=65532:65532 /out/data /data
VOLUME ["/data/out"]
ENTRYPOINT ["/etl", "-source", "/data/in", "-db", "/data/out/etl.db"]
