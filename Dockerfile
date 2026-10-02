# Base images are pinned by digest for reproducible builds; Dependabot keeps them current.

FROM golang:1.26@sha256:0f063af2d465d8dcae54cce04278ada488b96f77b42449c8d071e47d016cc65a AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# modernc.org/sqlite is pure Go, so the binary is fully static and runs on distroless/alpine.
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/etl ./cmd/etl
# distroless has no shell to mkdir with; pre-create the output dir so prod can chown it.
RUN mkdir -p /out/data/out

# Local development only (never deploy): has a shell and the sqlite3 CLI so you can exec in and inspect data.
FROM alpine:3.22@sha256:5291449c3df73caf6ed85e649dec1b9e818b39a5d8c871e97afc13e9cd5e8fa8 AS dev
RUN apk add --no-cache sqlite
COPY --from=build /out/etl /usr/local/bin/etl
RUN mkdir -p /data/in /data/out
WORKDIR /data
ENTRYPOINT ["etl", "-source", "/data/in", "-db", "/data/out/etl.db"]

# Production image (default target): no shell, runs as non-root.
FROM gcr.io/distroless/static-debian12:nonroot@sha256:afa5c872c891853ca7fcf1f12c3edb23f7eeef36189728842dd51042ff57f7ab AS prod
COPY --from=build /out/etl /etl
# Owned by nonroot so a fresh named volume inherits writable ownership.
COPY --from=build --chown=65532:65532 /out/data /data
VOLUME ["/data/out"]
ENTRYPOINT ["/etl", "-source", "/data/in", "-db", "/data/out/etl.db"]
