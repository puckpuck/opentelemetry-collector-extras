IMAGE_REPOSITORY ?= "ghcr.io/puckpuck/opentelemetry-collector-extras/otelcol-custom"
IMAGE_TAG ?= "latest"

.PHONY: install-builder
install-builder:
	go install go.opentelemetry.io/collector/cmd/builder@latest
	go install go.opentelemetry.io/collector/cmd/mdatagen@latest

.PHONY: build-local
build-local:
	builder --config=ocb.yaml

.PHONY: build
build:
	docker build -t $(IMAGE_REPOSITORY):$(IMAGE_TAG) .

.PHONY: build-and-push
build-and-push: build
	docker push $(IMAGE_REPOSITORY):$(IMAGE_TAG)

.PHONY: build-and-push-multiarch
build-and-push-multiarch:
	docker buildx build --push --platform linux/amd64,linux/arm64 -t $(IMAGE_REPOSITORY):$(IMAGE_TAG) .

.PHONY: run
run:
	./dist/otelcol-custom --config=config.yaml

.PHONY: debug
debug:
	dlv --listen=:2345 --headless=true --api-version=2 --accept-multiclient --log exec ./dist/otelcol-custom -- --config=config.yaml

