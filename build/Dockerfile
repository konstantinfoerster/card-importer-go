FROM golang:1.24-alpine3.21 AS builder

WORKDIR /app

# download go modules
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# copy source files
COPY cmd /app/cmd
COPY internal /app/internal

ENV GOOS="linux"
ENV GOARCH="amd64"
ENV CGO_ENABLED="0"

RUN go build -ldflags="-s -w" -o card-dataset-cli cmd/dataset/main.go && \
    chmod 0755 /app/card-dataset-cli && \
    go build -ldflags="-s -w" -o card-images-cli cmd/images/main.go && \
    chmod 0755 /app/card-images-cli

FROM alpine:3.21 AS dev

ARG RELEASE
ENV IMG_VERSION="${RELEASE}"

COPY --from=builder /app/card-dataset-cli /usr/bin/
COPY --from=builder /app/card-images-cli /usr/bin/

# hadolint ignore=DL3018
RUN set -eux; \
    apk add --no-progress --quiet --no-cache --upgrade \
    tzdata

USER nobody

CMD ["/usr/bin/card-dataset-cli", "--config", "/config/application.yaml"]

LABEL org.opencontainers.image.title="Card Importer CLI" \
      org.opencontainers.image.description="CLI tool to import card datasets into a database" \
      org.opencontainers.image.version="$IMG_VERSION" \
      org.opencontainers.image.source="https://github.com/konstantinfoerster/card-importer-go.git" \
      org.opencontainers.image.vendor="Konstantin Förster" \
      org.opencontainers.image.authors="Konstantin Förster"

FROM scratch AS prod

ARG RELEASE
ENV IMG_VERSION="${RELEASE}"

COPY --from=dev /usr/bin/card-dataset-cli /usr/bin/card-dataset-cli
COPY --from=dev /usr/bin/card-images-cli /usr/bin/card-images-cli

USER nobody

CMD ["/usr/bin/card-dataset-cli", "--config", "/config/application.yaml"]

LABEL org.opencontainers.image.title="Card Importer CLI" \
      org.opencontainers.image.description="CLI tool to import card datasets into a database" \
      org.opencontainers.image.version="$IMG_VERSION" \
      org.opencontainers.image.source="https://github.com/konstantinfoerster/card-importer-go.git" \
      org.opencontainers.image.vendor="Konstantin Förster" \
      org.opencontainers.image.authors="Konstantin Förster"
