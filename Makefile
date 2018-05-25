SUBSCRIBER_BINARY_NAME = subscriber
GATEWAY_BINARY_NAME = gateway

SUBSCRIBER_CMD_PATH = cmd/subscriber
GATEWAY_CMD_PATH = cmd/gateway

GATEWAY_IMG ?= somebody/brigade-eventgrid-gateway
SUBSCRIBER_IMG ?= somebody/brigade-eventgrid-subscriber

TAG ?= edge

OUTPUT_DIR = bin

.PHONY: linux
linux:
	$(MAKE) subscriber-linux && \
	$(MAKE) gateway-linux

.PHONY: subscriber-linux
subscriber-linux:
	cd $(SUBSCRIBER_CMD_PATH) && \
	GOOS=linux go build -o ../../$(OUTPUT_DIR)/$(SUBSCRIBER_BINARY_NAME)

.PHONY: gateway-linux
gateway-linux:
	cd $(GATEWAY_CMD_PATH) && \
	GOOS=linux go build -o ../../$(OUTPUT_DIR)/$(GATEWAY_BINARY_NAME)

.PHONY: docker-build
docker-build: linux
docker-build:
	docker build -f Dockerfile.gateway -t $(GATEWAY_IMG):$(TAG) .
	docker build -f Dockerfile.subscriber -t $(SUBSCRIBER_IMG):$(TAG) .

.PHONY: test
test:
	go test ./pkg/... ./cmd/...
