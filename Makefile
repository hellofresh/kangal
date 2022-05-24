NO_COLOR=\033[0m
OK_COLOR=\033[32;01m
ERROR_COLOR=\033[31;01m
WARN_COLOR=\033[33;01m

VERSION ?= "0.0.0-dev-$(shell git rev-parse --short HEAD)"

GOHOSTOS=$(shell go env GOHOSTOS)
GOHOSTARCH=$(shell go env GOHOSTARCH)

.PHONY: all clean update-codegen verify-codegen test-unit build

all: clean update-codegen verify-codegen test-unit build

# Cleans our project: deletes binaries
clean:
	@printf "$(OK_COLOR)==> Cleaning project$(NO_COLOR)\n"
	@if [ -d bin ] ; then rm -rf bin/* ; fi
	@if [ -d tools/bin ] ; then rm -rf tools/bin ; fi
	@if [ -d tools/vendor ] ; then rm -rf tools/vendor ; fi

# Runs unit-tests
test-unit:
	@printf "$(OK_COLOR)==> Running unit tests$(NO_COLOR)\n"
	@CGO_ENABLED=0 go test -short ./...

# Runs integration-tests
test-integration:
	@printf "$(OK_COLOR)==> Running integration tests$(NO_COLOR)\n"
	@CGO_ENABLED=1 go test -race -p=1 -cover ./... -coverprofile=coverage.txt -covermode=atomic

# Builds the project
build:
	@printf "$(OK_COLOR)==> Building v${VERSION}$(NO_COLOR)\n"
	@CGO_ENABLED=0 go build -ldflags "-s -w" -ldflags "-X main.version=${VERSION}" -o "bin/kangal"

# Apply CRD
apply-crd:
	@printf "$(OK_COLOR)==> Applying Kangal CRD to the current cluster $(NO_COLOR)\n"
	@kubectl delete crd loadtests.kangal.hellofresh.com || true
	@kubectl apply -f charts/kangal/crds/loadtest.yaml

dev-lint:
	@printf "$(OK_COLOR)==> Linting code$(NO_COLOR)\n"
	@docker run --rm -v $(CURDIR):/app -w /app golangci/golangci-lint:v1.42.1 golangci-lint run -v

update-codegen:
	@printf "$(OK_COLOR)==> Running codegen update$(NO_COLOR)\n"
	@go mod vendor
	@./hack/update-codegen.sh
	@rm -rf _tmp

verify-codegen:
	@printf "$(OK_COLOR)==> Running codegen verification$(NO_COLOR)\n"
	@go mod vendor
	@./hack/verify-codegen.sh

