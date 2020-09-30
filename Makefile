NO_COLOR=\033[0m
OK_COLOR=\033[32;01m
ERROR_COLOR=\033[31;01m
WARN_COLOR=\033[33;01m

VERSION ?= "0.0.0-dev-$(shell git rev-parse --short HEAD)"

.PHONY: all clean test-unit build

all: clean test-unit build

# Cleans our project: deletes binaries
clean:
	@printf "$(OK_COLOR)==> Cleaning project$(NO_COLOR)\n"
	@if [ -d bin ] ; then rm -rf bin/* ; fi

# Runs unit-tests
test-unit:
	@printf "$(OK_COLOR)==> Running unit tests$(NO_COLOR)\n"
	@CGO_ENABLED=0 go test -short ./...

# Runs integration-tests
test-integration:
	@printf "$(OK_COLOR)==> Running integration tests$(NO_COLOR)\n"
	@CGO_ENABLED=1 go test -p=1 -cover ./...

# Builds the project
build:
	@printf "$(OK_COLOR)==> Building v${VERSION}$(NO_COLOR)\n"
	@CGO_ENABLED=0 go build -ldflags "-s -w" -ldflags "-X main.version=${VERSION}" -o "bin/kangal"

# Apply CRD
apply-crd:
	@printf "$(OK_COLOR)==> Applying Kangal CRD to the current cluster $(NO_COLOR)\n"
	@kubectl delete crd loadtests.kangal.hellofresh.com || true
	@kubectl apply -f charts/kangal/crd.yaml
