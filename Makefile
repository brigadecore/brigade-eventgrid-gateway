SUBSCRIBER_BINARY_NAME = subscriber
GATEWAY_BINARY_NAME = gateway

SUBSCRIBER_CMD_PATH = cmd/subscriber
GATEWAY_CMD_PATH = cmd/gateway

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
