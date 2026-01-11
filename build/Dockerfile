FROM golang:1.25-alpine3.23 AS builder

ENV GOOS="linux" \
    GOARCH="amd64" \
    CGO_ENABLED="0"

WORKDIR /app

# download go modules
COPY go.mod go.sum ./
RUN go mod download && go mod verify 

# copy source files
COPY cmd /app/cmd
COPY internal /app/internal

RUN go build -tags timetzdata -ldflags="-s -w" -o card-dataset-cli cmd/dataset/main.go && \
  chmod 0755 /app/card-dataset-cli && \
  go build -tags timetzdata -ldflags="-s -w" -o card-images-cli cmd/images/main.go && \
  chmod 0755 /app/card-images-cli 

FROM alpine:3.23 AS dev

ARG RELEASE
ENV IMG_VERSION="${RELEASE}"

RUN addgroup -g 10001 nonroot \
  && adduser -D -u 10001 -G nonroot nonroot 

COPY --from=builder --chown=nonroot:nonroot /app/card-dataset-cli /usr/bin/
COPY --from=builder --chown=nonroot:nonroot /app/card-images-cli /usr/bin/

USER nonroot

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

USER 10001:10001

CMD ["/usr/bin/card-dataset-cli", "--config", "/config/application.yaml"]

LABEL org.opencontainers.image.title="Card Importer CLI" \
      org.opencontainers.image.description="CLI tool to import card datasets into a database" \
      org.opencontainers.image.version="$IMG_VERSION" \
      org.opencontainers.image.source="https://github.com/konstantinfoerster/card-importer-go.git" \
      org.opencontainers.image.vendor="Konstantin Förster" \
      org.opencontainers.image.authors="Konstantin Förster"
