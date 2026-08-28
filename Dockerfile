# syntax=docker/dockerfile:1

# ---- frontend ----
FROM node:20-alpine AS web-build
WORKDIR /web
COPY web/package.json web/package-lock.json* ./
RUN npm ci
COPY web/ ./
RUN npm run build

# ---- backend ----
# CGO is disabled deliberately: the SQLite driver (modernc.org/sqlite) is
# pure Go, so the binary doesn't need a C toolchain and stays trivially
# cross-compilable.
FROM golang:1.25-alpine AS go-build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/ cmd/
COPY cli/ cli/
COPY internal/ internal/
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/byteforge ./cmd/byteforge

# ---- runtime ----
FROM alpine:3.20
RUN adduser -D -u 10001 byteforge && mkdir -p /data && chown byteforge:byteforge /data

WORKDIR /app
COPY --from=go-build /out/byteforge /usr/local/bin/byteforge
COPY --from=web-build /web/dist ./web/dist

USER byteforge
EXPOSE 8080
VOLUME ["/data"]

HEALTHCHECK --interval=30s --timeout=3s --retries=3 \
    CMD wget -qO- http://localhost:8080/healthz || exit 1

ENTRYPOINT ["byteforge"]
CMD ["serve", "--addr", ":8080", "--db", "/data/byteforge.db", "--static", "/app/web/dist"]
