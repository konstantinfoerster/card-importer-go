BINARY_NAME_DATA=card-importer
BINARY_NAME_DATA=card-images
CURRENT_DIR=$(shell pwd)
ifndef VERSION
override VERSION = local-dev
endif

.PHONY: build-data
build-data:
	go build -o ${BINARY_NAME_DATA} cmd/data/main.go
.PHONY: build-image
build-image:
	go build -o ${BINARY_NAME_IMAGE} cmd/images/main.go
.PHONY: run-image
run-image:
	go run cmd/images/main.go -c configs/application-local.yaml
.PHONY: run-data
run-data:
	go run cmd/dataset/main.go -c configs/application-local.yaml
.PHONY: docker-dev
docker-dev:
	@echo "Build dev image version $(VERSION)"
	docker build --pull --build-arg RELEASE="$(VERSION)" -t card-importer-go:$(VERSION) --target dev -f build/Dockerfile .
.PHONY: docker-build
docker-build:
	@echo "Build prod image version $(VERSION)"
	docker build --pull --build-arg RELEASE="$(VERSION)" -t card-importer-go:$(VERSION) --target prod -f build/Dockerfile .
.PHONY: test-unit
test-unit:
	go test --short --count=1 ./...
.PHONY: test
test:
	go test --count=1 ./...
.PHONY: update
update:
	go get github.com/corona10/goimagehash 
	go get github.com/jackc/pgx/v5 
	go get github.com/rs/zerolog 
	go get github.com/stretchr/testify
	go get github.com/testcontainers/testcontainers-go 
	go get golang.org/x/sync 
	go get go.yaml.in/yaml/v3
	go mod tidy
.PHONY: lint
lint:
	docker run --pull always --rm -v ${CURRENT_DIR}\:/app -w /app golangci/golangci-lint\:v2.10-alpine golangci-lint run -v
	docker run --pull always --rm -i hadolint/hadolint < build/Dockerfile

