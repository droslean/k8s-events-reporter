.PHONY: all
all: check test build

.PHONY: check
check: ## Lint code
	gofmt -s -l $(shell go list -f '{{ .Dir }}' ./... ) | grep ".*\.go"; if [ "$$?" = "0" ]; then gofmt -s -d $(shell go list -f '{{ .Dir }}' ./... ); exit 1; fi
	go vet ./cmd/... ./pkg/...

format:
	gofmt -s -w $(shell go list -f '{{ .Dir }}' ./... )
.PHONY: format

.PHONY: build
build: ## Build binary
	go build -v -o events-reporter ./cmd/events-reporter

.PHONY: install
install: ## Install binary
	go install ./cmd/events-reporter

.PHONY: test
test: ## Run tests
	go test ./...


.PHONY: minikube-start
minikube-start: ## Start minikube
	minikube config set WantReportErrorPrompt false
	minikube start

.PHONY: build-image
build-image: ## Builds the events-reporter docker image
	eval $(minikube docker-env)
	docker build . -t events-reporter

.PHONY: deploy
deploy: ## Deploys the events-reporter
	kubectl create namespace events-reporter
	kubectl create cm events-reporter-config -n events-reporter --from-file static/config.yaml
	kubectl create -f deploy/events-viewer-rbac.yaml -n events-reporter
	kubectl create -f deploy/events-reporter-deployment.yaml -n events-reporter

.PHONY: start-deploy
start-deploy: build minikube-start build-image deploy ## Start minikube and deploy events-reporter

.PHONY: help
help:
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

