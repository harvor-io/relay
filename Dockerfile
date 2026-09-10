# syntax=docker/dockerfile:1

FROM golang:1.26-alpine AS builder

RUN apk add --no-cache git

WORKDIR /src

ENV CGO_ENABLED=0

COPY go.mod go.sum ./

RUN --mount=type=cache,id=gomod,target=/go/pkg/mod \
    go mod download

COPY . .

ARG VERSION=dev

RUN --mount=type=cache,id=gomod,target=/go/pkg/mod \
    --mount=type=cache,id=gobuild,target=/root/.cache/go-build \
    go install -trimpath -ldflags "-s -w -X main.version=${VERSION}" ./cmd/relay

FROM alpine:3.24 AS runner

RUN apk add --no-cache ca-certificates \
 && adduser -S -D -H -u 10001 appuser

COPY --from=builder /go/bin/relay /usr/local/bin/relay

# A writable working directory for the non-root user. relay resolves any
# relative paths (and, by default, its SQLite file) against this directory;
# mount a volume here to persist that state.
RUN mkdir -p /app && chown 10001 /app
WORKDIR /app

USER 10001

EXPOSE 8080

# Must name a subcommand: bare `relay` is the help action, which exits 0.
CMD ["relay", "serve"]
