[![Pipeline](https://github.com/konstantinfoerster/card-importer-go/actions/workflows/pipeline.yml/badge.svg?branch=main)](https://github.com/konstantinfoerster/card-importer-go/actions/workflows/pipeline.yml)

# CLI Card Importer

Command line application that downloads the latest card data set from [mtgjson.com](https://mtgjson.com/) and imports it
into a local postgres database.

## Run locally

### Database

First start a local postgres database e.g. via `docker-compose up -d`. This will start the database on port **15432**.
Use `docker-compose pull` to update the image version.

### Import Dataset

Run `go run cmd/dataset/main.go` to start the tool with the default configuration file (configs/application.yaml).

Optional parameter:

| Flag            | Usage                              | Default Value            | Description                                                      |
|-----------------|------------------------------------|--------------------------|------------------------------------------------------------------|
| `-c`,`--config` | `-c configs/application-prod.yaml` | configs/application.yaml | path to the config file                                          |
| `-u`,`--url`    | `-u https://localhost/dataset.zip` | not set                  | dataset download url as json or zip file                         |
| `-f`,`--file`   | `-f ./dataset.json`                | not set                  | dataset as json file, has precedence over the url flag or config |

### Import Images

Run `go run cmd/images/main.go` to start the tool with the default configuration file (configs/application.yaml).

Optional parameter:

| Flag            | Usage                              | Default Value            | Description                |
|-----------------|------------------------------------|--------------------------|----------------------------|
| `-c`,`--config` | `-c configs/applicationLocal.yaml` | configs/application.yaml | path to the config file    | 
| `-p`,`--page`   | `-p 21`                            | 1                        | start page number          |
| `-s`,`--size`   | `-s 100`                           | 20                       | amount of entries per page |

## Test

* Run **all** tests with `go test ./...`
* Run **unit tests** `go test -short ./...`
* Run **integration tests** `go test -run Integration ./...`

**Integration tests** require **docker** to be installed.

## Build

### Import Dataset

Build it with `go build -o card-dataset-cli cmd/dataset/main.go`

### Import Images

Build it with `go build -o card-images-cli cmd/images/main.go`

## Dependencies

Update all dependencies with `go get -u ./...`. Run `go mod tidy` afterwards to update and cleanup the `go.mod` file.
For mor information check: https://github.com/golang/go/wiki/Modules#how-to-upgrade-and-downgrade-dependencies

## Misc

### Docker linting

The linter [Hadolint](https://github.com/hadolint/hadolint) can be used to apply best practice on your Dockerfile.

Just run `docker run --rm -i hadolint/hadolint < Dockerfile` to check your Dockerfile.

### Golang linting

The lint aggregator [golangci-lint](https://golangci-lint.run/) can be used to apply best practice and find errors in
your golang code.

Just run `docker run --rm -v $(pwd):/app -w /app golangci/golangci-lint:v1.45.2 golangci-lint run -v` inside the root
dir of the project to start the linting process.

## TODOS

* Provide a make file
    * https://earthly.dev/blog/golang-makefile/

First idea how to serve the images:

1. Download all images via this tool
2. Compress everything into one zip file
3. Upload to a server to host the files
    1. Maybe use Google Drive (got 15GB for free and the api can be accessed via
       golang https://developers.google.com/drive/api/quickstart/go)
4. Build container that is able to server the images from file system
    1. Startup process of the container downloads the hosted compressed file
    2. Extract all images into served dir
