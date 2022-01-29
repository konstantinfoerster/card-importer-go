[![Pipeline](https://github.com/konstantinfoerster/card-importer-go/actions/workflows/pipeline.yml/badge.svg?branch=main)](https://github.com/konstantinfoerster/card-importer-go/actions/workflows/pipeline.yml)

# CLI Card Importer

Command line application that downloads the latest card data set from [mtgjson.com](https://mtgjson.com/) and imports it
into a local postgres database.

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
For mor information check: https://github.com/golang/go/wiki/Modules#how-to-upgrade-and-downgrade-dependencies

## Misc

### Docker linting

The linter [Hadolint](https://github.com/hadolint/hadolint) can be used to apply best practice on your Dockerfile.

Just run `docker run --rm -i hadolint/hadolint < Dockerfile` to check your Dockerfile.

### Golang linting

The lint aggregator [golangci-lint](https://golangci-lint.run/) can be used to apply best practice and find errors in
your golang code.

Just run `docker run --rm -v $(pwd):/app -w /app golangci/golangci-lint:v1.43.0 golangci-lint run -v` inside the root
dir of the project to start the linting process.

## TODOS

* Maybe drop the internal package?
* Provide a make file
    * https://earthly.dev/blog/golang-makefile/
* Persist NULL instead of empty string if column is nullable

## For later usage

Select card with all faces into as json:

```
SELECT to_jsonb(r) FROM ( SELECT c.id, c.card_set_code, c.number, c.border, c.rarity,  COALESCE(json_agg(t), '[]') as faces FROM  card AS c LEFT JOIN card_translation AS t ON c.id = t.card_id  GROUP BY c.id) r
```
