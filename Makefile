VERSION ?= $(shell date +"%Y%m%d")
OUTPUT_DIR := _output

.PHONY: format
format:
	gofmt -w -s .

.PHONY: test
test:
	go test -v -race -timeout 3m ./...

# Install golangci-lint tool to run lint locally
# https://golangci-lint.run/usage/install
.PHONY: lint
lint:
	golangci-lint run

# clean the output directory
.PHONY: clean
clean:
	rm -rf "$(OUTPUT_DIR)"

.PHONY: build
build:
	go build -o _output/ndc-rest-schema .
	
# build the ndc-rest-schema for all given platform/arch
.PHONY: ci-build
ci-build: export CGO_ENABLED=0
ci-build: clean
	go get github.com/mitchellh/gox && \
	go run github.com/mitchellh/gox -ldflags '-X github.com/hasura/ndc-rest-schema/version.BuildVersion=$(VERSION) -s -w -extldflags "-static"' \
		-osarch="linux/amd64 darwin/amd64 windows/amd64 darwin/arm64" \
		-output="$(OUTPUT_DIR)/$(VERSION)/ndc-rest-schema-{{.OS}}-{{.Arch}}" \
		.