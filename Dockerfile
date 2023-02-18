##### BUILDER #####

FROM golang:1.20-alpine3.17 as builder

## Task: copy source files
COPY . /src
WORKDIR /src

## Task: fetch project deps
RUN go mod download

## Task: build project
ENV GOOS="linux"
ENV GOARCH="amd64"
ENV CGO_ENABLED="0"

RUN go build -ldflags="-s -w" -o card-dataset-cli cmd/dataset/main.go && \
    go build -ldflags="-s -w" -o card-images-cli cmd/images/main.go

## Task: set permissions
RUN chmod 0755 /src/card-dataset-cli && chmod 0755 /src/card-images-cli

##### TARGET #####

FROM alpine:3.17

ARG RELEASE
ENV IMG_VERSION="${RELEASE}"

COPY --from=builder /src/card-dataset-cli /usr/bin/
COPY --from=builder /src/card-images-cli /usr/bin/

# hadolint ignore=DL3018
RUN set -eux; \
    apk add --no-progress --quiet --no-cache --upgrade \
    tzdata

CMD ["/usr/bin/card-dataset-cli", "--config", "/config/application.yaml"]

LABEL org.opencontainers.image.title="Card Importer CLI" \
      org.opencontainers.image.description="CLI tool to import card datasets into a database" \
      org.opencontainers.image.version="$IMG_VERSION" \
      org.opencontainers.image.source="https://github.com/konstantinfoerster/card-importer-go.git" \
      org.opencontainers.image.vendor="Konstantin Förster" \
      org.opencontainers.image.authors="Konstantin Förster"


