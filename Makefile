.PHONY: build clean test deploy

GOOS=linux
GOARCH=arm64
CGO_ENABLED=0

# Build all Lambda binaries
build:
	@echo "Building interaction Lambda..."
	cd cmd/interaction && GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=$(CGO_ENABLED) go build -o bootstrap main.go
	@echo "Building worker Lambda..."
	cd cmd/worker && GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=$(CGO_ENABLED) go build -o bootstrap main.go
	@echo "Building scheduler Lambda..."
	cd cmd/scheduler && GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=$(CGO_ENABLED) go build -o bootstrap main.go

# Clean build artifacts
clean:
	rm -f cmd/interaction/bootstrap
	rm -f cmd/worker/bootstrap
	rm -f cmd/scheduler/bootstrap

# Run tests
test:
	go test -v -race ./...

# Deploy with SAM
deploy: build
	sam deploy --guided --template-file template.yaml

# Deploy without prompts (for CI/CD)
deploy-ci: build
	sam deploy --no-confirm-changeset --template-file template.yaml --stack-name discord-ops-bot

# Local testing
local-test:
	sam local start-api --template template.yaml
