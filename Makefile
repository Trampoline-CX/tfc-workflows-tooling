# General variables
BLUE_COLOR  := \033[36m
NO_COLOR    := \033[0m

LDFLAGS     := -X main.commitHash=${COMMIT_HASH} -X main.buildDate=${BUILD_DATE}
GOBUILD     := go build -ldflags "$(LDFLAGS)"

REPO_NAME := $(shell basename $(shell git rev-parse --show-toplevel))

.PHONY: fmt
fmt: log-fmt
	gofmt -s -l -w .

.PHONY: test
test: log-test
	go test ./... $(TESTARGS) -timeout 15m


.PHONY: build
build: GOOS   = $(shell go env GOOS)
build: GOARCH = $(shell go env GOARCH)
build: log-build ## Build binary for current OS/ARCH
	@ GOOS=$(GOOS) GOARCH=$(GOARCH) $(GOBUILD)

.PHONY: docker
docker: log-docker ## Build Docker image
	@ docker build -t $(REPO_NAME) .

.PHONY: docker-run
docker-run: log-docker-run ## Run the Docker container
	@ docker run -it --rm -e TF_API_TOKEN=${TF_API_TOKEN} -e "TF_CLOUD_ORGANIZATION=${TF_CLOUD_ORGANIZATION}" $(REPO_NAME)

.PHONY: help
.DEFAULT_GOAL := help
help:
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "$(BLUE_COLOR)%-20s$(NO_COLOR) %s\n", $$1, $$2}'

log-%:
	@grep -h -E '^$*:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "$(BLUE_COLOR)==> %s$(NO_COLOR)\n", $$2}'