ifndef LINT_MK
LINT_MK:=# Prevent repeated "-include".

GOLANGCI_LINT_BIN=./out/golangci-lint

include ./make/verbose.mk
include ./make/go.mk

.PHONY: lint
## Runs linters on Go code files and YAML files
lint: lint-go-code lint-yaml

YAML_FILES := $(shell find . -path ./vendor -prune -o -type f -regex ".*y[a]ml" -print)
.PHONY: lint-yaml
## runs yamllint on all yaml files
lint-yaml: ./vendor ${YAML_FILES}
	$(Q)yamllint -c .yamllint $(YAML_FILES)

.PHONY: lint-go-code
## Checks the code with golangci-lint
lint-go-code: ./vendor
	$(Q)GOCACHE=$(shell pwd)/out/gocache go get github.com/golangci/golangci-lint/cmd/golangci-lint
	$(Q)${GOPATH}/bin/golangci-lint ${V_FLAG} run

endif
