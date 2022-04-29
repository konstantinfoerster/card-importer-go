##### BUILDER #####

# there is no need to build everything again. we already have the alpine based image and can use its atifacts 

##### TARGET #####
ARG RELEASE

FROM card-importer-cli:${RELEASE} AS copy-src

FROM scratch

ARG RELEASE
ENV IMG_VERSION="${RELEASE}"

COPY --from=copy-src /usr/local/bin/card-dataset-cli /

ENTRYPOINT ["/card-dataset-cli"]
CMD ["--config", "/config/application.yaml"]

LABEL org.opencontainers.image.title="Card Importer CLI" \
      org.opencontainers.image.description="CLI tool to import card data sets into a database" \
      org.opencontainers.image.version="$IMG_VERSION" \
      org.opencontainers.image.source="https://github.com/konstantinfoerster/card-importer-go.git" \
      org.opencontainers.image.vendor="Konstantin Förster" \
      org.opencontainers.image.authors="Konstantin Förster"