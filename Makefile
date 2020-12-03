NO_COLOR=\033[0m
OK_COLOR=\033[32;01m
ERROR_COLOR=\033[31;01m
WARN_COLOR=\033[33;01m

VERSION ?= "0.0.0-dev-$(shell git rev-parse --short HEAD)"

GOHOSTOS=$(shell go env GOHOSTOS)
GOHOSTARCH=$(shell go env GOHOSTARCH)

TOOLS_BIN ?= $(CURDIR)/tools/bin
TOOLS_VENDOR ?= $(CURDIR)/tools/vendor

# Include tools
include $(CURDIR)/tools/tools.mk

.PHONY: all clean test-unit build

all: clean test-unit build

# Cleans our project: deletes binaries
clean:
	@printf "$(OK_COLOR)==> Cleaning project$(NO_COLOR)\n"
	@if [ -d bin ] ; then rm -rf bin/* ; fi
	@if [ -d tools/bin ] ; then rm -rf tools/bin ; fi
	@if [ -d tools/vendor ] ; then rm -rf tools/vendor ; fi

# Runs unit-tests
test-unit: protoc
	@printf "$(OK_COLOR)==> Running unit tests$(NO_COLOR)\n"
	@CGO_ENABLED=0 go test -short ./...

# Runs integration-tests
test-integration: protoc
	@printf "$(OK_COLOR)==> Running integration tests$(NO_COLOR)\n"
	@CGO_ENABLED=1 go test -race -p=1 -cover ./... -coverprofile=coverage.txt -covermode=atomic

# Builds the project
build: protoc
	@printf "$(OK_COLOR)==> Building v${VERSION}$(NO_COLOR)\n"
	@CGO_ENABLED=0 go build -ldflags "-s -w" -ldflags "-X main.version=${VERSION}" -o "bin/kangal"

# Apply CRD
apply-crd:
	@printf "$(OK_COLOR)==> Applying Kangal CRD to the current cluster $(NO_COLOR)\n"
	@kubectl delete crd loadtests.kangal.hellofresh.com || true
	@kubectl apply -f charts/kangal/crd.yaml

# Transpile proto file(s) to source code
protoc: tools
	@printf "$(OK_COLOR)==> Compiling ProtoBuf$(NO_COLOR)\n"
	@$(tools/protoc) \
		--plugin=$(tools/protoc-gen-go) \
		--plugin=$(tools/protoc-gen-go-grpc) \
		--plugin=$(tools/protoc-gen-grpc-gateway) \
		--plugin=$(tools/protoc-gen-openapiv2) \
		--proto_path=$(tools/gateway)/third_party/googleapis \
		--proto_path=$(tools/protobuf)/src \
		--proto_path=$(CURDIR)/proto \
		--go_out=$(CURDIR)/pkg/proxy/rpc/pb \
		--go-grpc_out=$(CURDIR)/pkg/proxy/rpc/pb \
		--go-grpc_opt=require_unimplemented_servers=false \
		--grpc-gateway_out=$(CURDIR)/pkg/proxy/rpc/pb \
		--grpc-gateway_opt=logtostderr=true \
		--openapiv2_out=$(CURDIR)/pkg/proxy/rpc/pb \
		--openapiv2_opt=logtostderr=true \
		$(CURDIR)/proto/*/*/*/*.proto

# Download all tools required for development, testing and releasing
tools: $(tools/protoc) $(tools/protobuf) $(tools/protoc-gen-go) $(tools/protoc-gen-go-grpc) $(tools/protoc-gen-grpc-gateway) $(tools/protoc-gen-openapiv2) $(tools/gateway)
.PHONY: tools

dev-lint: protoc
	@printf "$(OK_COLOR)==> Linting code$(NO_COLOR)\n"
	@docker run --rm -v $(CURDIR):/app -w /app golangci/golangci-lint:v1.33.0 golangci-lint run -v

dev-buf: protoc
	@printf "$(OK_COLOR)==> Linting ProtoBuf$(NO_COLOR)\n"
	@docker run -it --rm -v $(CURDIR):/app -w /app bufbuild/buf:0.31.0 check lint --config=/app/.github/buf.yaml
