##### TARGET #####
ARG RELEASE

FROM card-importer-cli:${RELEASE} AS copy-src

FROM scratch

ARG RELEASE
ENV IMG_VERSION="${RELEASE}"

COPY --from=copy-src /usr/bin/card-dataset-cli /usr/bin/card-dataset-cli
COPY --from=copy-src /usr/bin/card-images-cli /usr/bin/card-images-cli

CMD ["/usr/bin/card-dataset-cli", "--config", "/config/application.yaml"]

LABEL org.opencontainers.image.title="Card Importer CLI" \
      org.opencontainers.image.description="CLI tool to import card datasets into a database" \
      org.opencontainers.image.version="$IMG_VERSION" \
      org.opencontainers.image.source="https://github.com/konstantinfoerster/card-importer-go.git" \
      org.opencontainers.image.vendor="Konstantin Förster" \
      org.opencontainers.image.authors="Konstantin Förster"