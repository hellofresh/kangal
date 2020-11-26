# protoc and source is used to generate code from a protobuf file and handle imports
PROTOC_VERSION=3.14.0
PROTOC_ZIP=protoc-$(PROTOC_VERSION)-linux-x86_64.zip
ifeq ($(GOHOSTOS)_$(GOHOSTARCH),darwin_amd64)
    PROTOC_ZIP=protoc-$(PROTOC_VERSION)-osx-x86_64.zip
endif
PROTOC_RELEASES_URI=https://github.com/protocolbuffers/protobuf/releases/download
PROTOC_DOWNLOAD=$(PROTOC_RELEASES_URI)/v$(PROTOC_VERSION)/$(PROTOC_ZIP)

tools/protoc = $(TOOLS_BIN)/protoc/${PROTOC_VERSION}/protoc
$(tools/protoc):
	@printf "$(OK_COLOR)==> Installing tools/protoc$(NO_COLOR)\n"
	mkdir -p "$(@D)"
	curl --fail --output "$@.zip" -L "$(PROTOC_DOWNLOAD)"
	go run tools/unzip.go "$@.zip" "$@" "bin/protoc"
	rm -f "$@.zip"

PROTOBUF_GZIP=protobuf-all-$(PROTOC_VERSION).tar.gz
PROTOBUF_DOWNLOAD=$(PROTOC_RELEASES_URI)/v$(PROTOC_VERSION)/$(PROTOBUF_GZIP)

tools/protobuf-src = $(TOOLS_VENDOR)/protobuf/${PROTOC_VERSION}/protobuf.tar.gz
$(tools/protobuf-src): $(tools/protoc)
	@printf "$(OK_COLOR)==> Installing tools/protobuf-src$(NO_COLOR)\n"
	mkdir -p "$(@D)"
	rm -rf $(TOOLS_VENDOR)/protobuf/current
	curl --fail --output "$@" -L "$(PROTOBUF_DOWNLOAD)"
	go run tools/untargzip.go "$@" "$(@D)"
	mv $(TOOLS_VENDOR)/protobuf/${PROTOC_VERSION}/protobuf-${PROTOC_VERSION} $(TOOLS_VENDOR)/protobuf/current

tools/protobuf = $(TOOLS_VENDOR)/protobuf/current
$(tools/protobuf): $(tools/protobuf-src)

# protoc-gen-go is the protoc plugin to generate golang protobuf code
GEN_GO_VERSION=$(shell grep google.golang.org/protobuf $(CURDIR)/go.mod | awk '{print $$2}')
GEN_GO_TGZ=protoc-gen-go.$(GEN_GO_VERSION).linux.amd64.tar.gz
ifeq ($(GOHOSTOS)_$(GOHOSTARCH),darwin_amd64)
    GEN_GO_TGZ=protoc-gen-go.$(GEN_GO_VERSION).darwin.amd64.tar.gz
endif
GEN_GO_RELEASES_URI=https://github.com/protocolbuffers/protobuf-go/releases/download
GEN_GO_DOWNLOAD=$(GEN_GO_RELEASES_URI)/$(GEN_GO_VERSION)/$(GEN_GO_TGZ)

tools/protoc-gen-go = $(TOOLS_BIN)/protoc-gen-go/${GEN_GO_VERSION}/protoc-gen-go
$(tools/protoc-gen-go): $(tools/protoc)
	@printf "$(OK_COLOR)==> Installing tools/protoc-gen-go$(NO_COLOR)\n"
	mkdir -p "$(@D)"
	curl --fail --output "$@.tar.gz" -L "$(GEN_GO_DOWNLOAD)"
	go run tools/untargzip.go "$@.tar.gz" "$(@D)" "protoc-gen-go"
	rm -f "$@.tar.gz"
	chmod +x "$@"

# protoc-gen-go-grpc is the protoc plugin to generate golang gRPC code
GEN_GO_GRPC_VERSION=$(shell grep google.golang.org/grpc/cmd/protoc-gen-go-grpc $(CURDIR)/go.mod | awk '{print $$2}')
GEN_GO_GRPC_TGZ=protoc-gen-go-grpc.$(GEN_GO_GRPC_VERSION).linux.386.tar.gz
ifeq ($(GOHOSTOS)_$(GOHOSTARCH),darwin_amd64)
    GEN_GO_GRPC_TGZ=protoc-gen-go-grpc.$(GEN_GO_GRPC_VERSION).darwin.amd64.tar.gz
endif
GEN_GO_GRPC_RELEASES_URI=https://github.com/grpc/grpc-go/releases/download
GEN_GO_GRPC_DOWNLOAD=$(GEN_GO_GRPC_RELEASES_URI)/cmd%2Fprotoc-gen-go-grpc%2F$(GEN_GO_GRPC_VERSION)/$(GEN_GO_GRPC_TGZ)

tools/protoc-gen-go-grpc = $(TOOLS_BIN)/protoc-gen-go-grpc/${GEN_GO_GRPC_VERSION}/protoc-gen-go-grpc
$(tools/protoc-gen-go-grpc): $(tools/protoc)
	@printf "$(OK_COLOR)==> Installing tools/protoc-gen-go-grpc$(NO_COLOR)\n"
	mkdir -p "$(@D)"
	curl --fail --output "$@.tar.gz" -L "$(GEN_GO_GRPC_DOWNLOAD)"
	go run tools/untargzip.go "$@.tar.gz" "$(@D)" "protoc-gen-go-grpc"
	rm -f "$@.tar.gz"
	chmod +x "$@"

# grpc-gateway is the protoc plugins set to generate grpc/rest gateway code
GRPC_GATEWAY_VERSION=$(shell grep github.com/grpc-ecosystem/grpc-gateway/v2 $(CURDIR)/go.mod | awk '{print $$2}')
GRPC_GATEWAY_GW_ASSET=protoc-gen-grpc-gateway-$(GRPC_GATEWAY_VERSION)-linux-x86_64
GRPC_GATEWAY_OA_ASSET=protoc-gen-openapiv2-$(GRPC_GATEWAY_VERSION)-linux-x86_64
ifeq ($(GOHOSTOS)_$(GOHOSTARCH),darwin_amd64)
    GRPC_GATEWAY_GW_ASSET=protoc-gen-grpc-gateway-$(GRPC_GATEWAY_VERSION)-darwin-x86_64
    GRPC_GATEWAY_OA_ASSET=protoc-gen-openapiv2-$(GRPC_GATEWAY_VERSION)-darwin-x86_64
endif
GRPC_GATEWAY_URI=https://github.com/grpc-ecosystem/grpc-gateway/releases/download
GRPC_GATEWAY_GW_DOWNLOAD=$(GRPC_GATEWAY_URI)/$(GRPC_GATEWAY_VERSION)/$(GRPC_GATEWAY_GW_ASSET)
GRPC_GATEWAY_OA_DOWNLOAD=$(GRPC_GATEWAY_URI)/$(GRPC_GATEWAY_VERSION)/$(GRPC_GATEWAY_OA_ASSET)

tools/protoc-gen-grpc-gateway = $(TOOLS_BIN)/grpc-gateway/${GRPC_GATEWAY_VERSION}/protoc-gen-grpc-gateway
$(tools/protoc-gen-grpc-gateway): $(tools/protoc)
	@printf "$(OK_COLOR)==> Installing tools/protoc-gen-grpc-gateway$(NO_COLOR)\n"
	mkdir -p "$(@D)"
	curl --fail --output "$@" -L "$(GRPC_GATEWAY_GW_DOWNLOAD)"
	chmod +x "$@"

tools/protoc-gen-openapiv2 = $(TOOLS_BIN)/grpc-gateway/${GRPC_GATEWAY_VERSION}/protoc-gen-openapiv2
$(tools/protoc-gen-openapiv2): $(tools/protoc)
	mkdir -p "$(@D)"
	curl --fail --output "$@" -L "$(GRPC_GATEWAY_OA_DOWNLOAD)"
	chmod +x "$@"

GRPC_GATEWAY_PROTO_URI=https://github.com/grpc-ecosystem/grpc-gateway/archive
GRPC_GATEWAY_PROTO_DOWNLOAD=$(GRPC_GATEWAY_PROTO_URI)/$(GRPC_GATEWAY_VERSION).tar.gz

tools/gateway-src = $(TOOLS_VENDOR)/gateway/${GRPC_GATEWAY_VERSION}/gateway.tar.gz
$(tools/gateway-src): $(tools/protoc-gen-grpc-gateway) $(tools/protoc-gen-openapiv2)
	@printf "$(OK_COLOR)==> Installing tools/gateway-src$(NO_COLOR)\n"
	mkdir -p "$(@D)"
	rm -rf $(TOOLS_VENDOR)/gateway/current
	curl --fail --output "$@" -L "$(GRPC_GATEWAY_PROTO_DOWNLOAD)"
	go run tools/untargzip.go "$@" "$(@D)"
	mv $(TOOLS_VENDOR)/gateway/${GRPC_GATEWAY_VERSION}/grpc-gateway-$(subst v,,$(GRPC_GATEWAY_VERSION)) $(TOOLS_VENDOR)/gateway/current

tools/gateway = $(TOOLS_VENDOR)/gateway/current
$(tools/gateway): $(tools/gateway-src)
