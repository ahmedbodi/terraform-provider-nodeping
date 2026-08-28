.PHONY: build test lint clean docker-build docker-test docker-tidy

DOCKER_IMAGE := terraform-provider-nodeping
GO_VERSION := 1.25

build:
	docker run --rm -v $(PWD):/app -w /app golang:$(GO_VERSION)-alpine sh -c "go mod tidy && CGO_ENABLED=0 go build -o terraform-provider-nodeping ."

test:
	docker run --rm -v $(PWD):/app -w /app --env-file .env golang:$(GO_VERSION)-alpine sh -c "go mod tidy && go test -v ./..."

test-unit:
	docker run --rm -v $(PWD):/app -w /app golang:$(GO_VERSION)-alpine sh -c "go mod tidy && go test -v ./internal/client/..."

tidy:
	docker run --rm -v $(PWD):/app -w /app golang:$(GO_VERSION)-alpine sh -c "go mod tidy"

lint:
	docker run --rm -v $(PWD):/app -w /app golangci/golangci-lint:latest golangci-lint run

fmt:
	docker run --rm -v $(PWD):/app -w /app golang:$(GO_VERSION)-alpine sh -c "gofmt -w ."

docker-build:
	docker build --target builder -t $(DOCKER_IMAGE):builder .
	docker build --target runtime -t $(DOCKER_IMAGE):latest .

docker-test:
	docker build --target test -t $(DOCKER_IMAGE):test .
	docker run --rm --env-file .env $(DOCKER_IMAGE):test

clean:
	rm -f terraform-provider-nodeping
	docker rmi $(DOCKER_IMAGE):builder $(DOCKER_IMAGE):latest $(DOCKER_IMAGE):test 2>/dev/null || true

help:
	@echo "Available targets:"
	@echo "  build       - Build the provider binary using Docker"
	@echo "  test        - Run all tests using Docker"
	@echo "  test-unit   - Run unit tests only using Docker"
	@echo "  tidy        - Run go mod tidy using Docker"
	@echo "  lint        - Run golangci-lint using Docker"
	@echo "  fmt         - Format Go code using Docker"
	@echo "  docker-build - Build Docker images"
	@echo "  docker-test  - Run tests in Docker container"
	@echo "  clean       - Remove build artifacts and Docker images"
