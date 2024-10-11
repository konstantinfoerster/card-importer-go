BINARY_NAME_DATA=card-importer
BINARY_NAME_DATA=card-images
CURRENT_DIR=$(shell pwd)

build-data:
	go build -o ${BINARY_NAME_DATA} cmd/data/main.go
build-image:
	go build -o ${BINARY_NAME_IMAGE} cmd/images/main.go
run-image:
	go run cmd/images/main.go -c configs/application-local.yaml
run-data:
	go run cmd/dataset/main.go -c configs/application-local.yaml
docker-dev:
	docker build -t card-importer:local --target dev -f build/Dockerfile .
docker-prod:
	docker build -t card-importer:local --target prod -f build/Dockerfile .
test-unit:
	go test --short --count=1 ./...
test:
	go test --count=1 ./...
update:
	go get -u ./...
	go mod tidy
lint:
	docker run --pull always --rm -v ${CURRENT_DIR}\:/app -w /app golangci/golangci-lint\:latest golangci-lint run -v
	docker run --pull always --rm -i hadolint/hadolint < build/Dockerfile


