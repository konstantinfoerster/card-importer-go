[![Pipeline](https://github.com/konstantinfoerster/card-importer-go/actions/workflows/pipeline.yml/badge.svg?branch=main)](https://github.com/konstantinfoerster/card-importer-go/actions/workflows/pipeline.yml)

# CLI Card Importer
Command line application that downloads the latest card data set from [mtgjson.com](https://mtgjson.com/) and imports it into a local postgres database. 

## Run locally
First start a local postgres database e.g. via `docker-compose up`. This will start the database on port **15432**.
Run `go run cmd/importer.go` to start the application.

## Configuration
A configuration file can be provided via the `-c` flag. If no flag is provided the default configuration will be used.
It can be found at **configs/application.yaml**.

## Test
* Run **all** tests with `go test ./...`
* Run **unit tests** `go test -short ./...`
* Run **integration tests** `go test -run Integration ./...`

**Integration tests** require **docker** to be installed.

## Build
Build with `go build cmd/importer.go`

## Dependencies
Update all dependencies with `go get -u ./...`. Run `go mod tidy` afterwards to update and cleanup the `go.mod` file.

## Misc
The linter [Hadolint](https://github.com/hadolint/hadolint) can be used to apply best practice on your Dockerfile.
Just run `docker run --rm -i hadolint/hadolint < Dockerfile` to check your Dockerfile.

## TODOS
* Maybe drop the internal package?
* Provide a make file