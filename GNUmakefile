NAME=tart
BINARY=packer-plugin-${NAME}

COUNT?=1
TEST?=$(shell go list ./...)
HASHICORP_PACKER_PLUGIN_SDK_VERSION?=$(shell go list -m github.com/hashicorp/packer-plugin-sdk | cut -d " " -f2)

.PHONY: build
build: PLUGIN_VERSION=$(shell git describe --tags --dirty --always | \
		sed 's/^v//' | perl -pe 's/(\d+\.\d+\.)(\d+)-.*/$${1}.($$2+1)."-dev"/e')
build:
	@go build -ldflags="-X '${BINARY}/version.Version=$(PLUGIN_VERSION)'" -o ${BINARY}

.PHONY: install
install: build
	@packer plugins install --path ${BINARY} "github.com/cirruslabs/$(NAME)"

dev: install

test:
	@go test -race -count $(COUNT) $(TEST) -timeout=3m

install-packer-sdc: ## Install packer sofware development command
	@go install github.com/hashicorp/packer-plugin-sdk/cmd/packer-sdc@${HASHICORP_PACKER_PLUGIN_SDK_VERSION}

plugin-check: install-packer-sdc build
	@packer-sdc plugin-check ${BINARY}

testacc: dev
	@PACKER_ACC=1 go test -count $(COUNT) -v $(TEST) -timeout=120m

generate: install-packer-sdc
	@go generate ./...
	@if [ -d ".docs" ]; then rm -r ".docs"; fi
	@packer-sdc renderdocs -src "docs" -partials docs-partials/ -dst ".docs/"
	@./.web-docs/scripts/compile-to-webdocs.sh "." ".docs" ".web-docs" "cirruslabs"
	@rm -r ".docs"
