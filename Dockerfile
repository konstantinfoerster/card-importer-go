##### BUILDER #####

FROM golang:1.17.9-alpine3.15 as builder

## Task: copy source files
COPY . /src
WORKDIR /src

## Task: fetch project deps
RUN go mod download

## Task: build project
ENV GOOS="linux"
ENV GOARCH="amd64"
ENV CGO_ENABLED="0"

RUN go build -ldflags="-s -w" -o card-dataset-cli cmd/dataset/main.go
RUN go build -ldflags="-s -w" -o card-images-cli cmd/images/main.go

## Task: set permissions
RUN chmod 0755 /src/card-dataset-cli && chmod 0755 /src/card-images-cli

# hadolint ignore=DL3018
RUN set -eux; \
    apk add --no-progress --quiet --no-cache --upgrade --virtual .run-deps \
    tzdata

# hadolint ignore=DL4006,SC2183
RUN set -eu +x; \
    printf '%30s\n' | tr ' ' -; \
    echo "RUNTIME DEPENDENCIES"; \
    PKGNAME=$(apk info --depends .rundeps \
        | sed '/^$/d;/depends/d' \
        | sed -r 's/^(.*)\~.*/\1/g' \
        | sort -u ); \
    printf '%s\n' "${PKGNAME}" \
        | while IFS= read -r pkg; do \
                apk info --quiet --description --no-network "${pkg}" \
                | sed -n '/description/p' \
                | grep -v gettext-tiny \
                | sed -r "s/($(echo "${pkg}" | sed -r 's/\+/\\+/g'))-(.*)\s.*/\1=\2/"; \
                done \
        | tee -a /usr/share/rundeps; \
    printf '%30s\n' | tr ' ' -

##### TARGET #####

FROM alpine:3.15

ARG RELEASE
ENV IMG_VERSION="${RELEASE}"

COPY --from=builder /src/card-dataset-cli /usr/bin/
COPY --from=builder /src/card-images-cli /usr/bin/
COPY --from=builder /usr/share/rundeps /usr/share/rundeps

RUN set -eux; \
    xargs -a /usr/share/rundeps apk add --no-progress --quiet --no-cache --upgrade --virtual .run-deps

CMD ["/usr/bin/card-dataset-cli", "--config", "/config/application.yaml"]

LABEL org.opencontainers.image.title="Card Importer CLI" \
      org.opencontainers.image.description="CLI tool to import card datasets into a database" \
      org.opencontainers.image.version="$IMG_VERSION" \
      org.opencontainers.image.source="https://github.com/konstantinfoerster/card-importer-go.git" \
      org.opencontainers.image.vendor="Konstantin Förster" \
      org.opencontainers.image.authors="Konstantin Förster"


