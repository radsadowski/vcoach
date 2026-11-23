FROM golang:1.26-alpine AS build
WORKDIR /src

# Dependencies are vendored (go mod vendor), so the build needs no network.
COPY . .
RUN CGO_ENABLED=0 go build -mod=vendor -trimpath -o /out/vcoach ./cmd/server

FROM alpine:3.21
COPY --from=build /out/vcoach /usr/local/bin/vcoach

EXPOSE 8080
USER nobody
ENTRYPOINT ["/usr/local/bin/vcoach"]
