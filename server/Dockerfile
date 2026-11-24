# syntax=docker/dockerfile:1
FROM golang:1.26-alpine AS build
WORKDIR /src

# Module cache is a BuildKit cache mount: downloaded once, reused across
# builds (and across dependency changes when go.mod is unchanged).
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -trimpath -o /out/vcoach ./cmd/server

FROM alpine:3.21
COPY --from=build /out/vcoach /usr/local/bin/vcoach

EXPOSE 8080
USER nobody
ENTRYPOINT ["/usr/local/bin/vcoach"]
