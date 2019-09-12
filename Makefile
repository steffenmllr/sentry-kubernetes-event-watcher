.PHONY: docker
docker: ## Builds the container
	docker build -t steffenmllr/sentry-kubernetes-event-watcher .

.PHONY: fmt
fmt: ## Run goimports on all go files
	find . -name '*.go' -not -wholename './vendor/*' | while read -r file; do goimports -w "$$file"; done

.PHONY: lint
lint: ## Run all the linters
	golangci-lint run --enable gofmt

.PHONY: build
build: ## Build a version
	go build -o ./watcher

.PHONY: run
run: ## Build a version
	go run main.go

.PHONY: clean
clean: ## Remove temporary files
	find . -type f -name ".gometalinter*" -delete
	find . -type f -name ".golangci-lint*" -delete
	go clean

# Absolutely awesome: http://marmelab.com/blog/2016/02/29/auto-documented-makefile.html
.PHONY: help
help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := build
