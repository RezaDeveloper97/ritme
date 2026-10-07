# syntax=docker/dockerfile:1
# Ritme Go API. Build context: backend-go/.
#   docker build -t ritme-backend-go:dev .

FROM golang:1.25-alpine AS build
WORKDIR /src
# proxy.golang.org answers 403 to some Iranian egress IPs; "|" falls through to the
# next source on ANY error (not only 404). go.sum still pins every checksum.
ARG GOPROXY="https://proxy.golang.org|https://goproxy.io|direct"
ENV CGO_ENABLED=0 GOOS=linux GOPROXY=${GOPROXY}
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

# alpine, not distroless: gcr.io answers 403 from Iranian IPs (the prod server and
# dev machines), Docker Hub does not. tzdata is embedded in the binary (time/tzdata).
FROM alpine:3.22
RUN apk add --no-cache ca-certificates
# Mount point of the media-storage volume (B-N8-02 lesson media): owned by 33:33 so Docker initialises the empty named
# volume with that owner, 0700 like the rest of the private storage.
RUN mkdir -p /var/lib/ritme-media && chown 33:33 /var/lib/ritme-media && chmod 0700 /var/lib/ritme-media
COPY --from=build /out/api /app/api
# uid/gid 33 = www-data in the PHP image: the Passport keys on the shared
# backend-storage volume are 0600 www-data, so any other uid cannot read them.
USER 33:33
ENV HTTP_ADDR=:80 \
    STORAGE_PATH=/var/www/html/storage
EXPOSE 80
HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=3 CMD ["/app/api", "healthcheck"]
ENTRYPOINT ["/app/api"]
