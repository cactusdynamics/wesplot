.PHONY: clean backend-lint lint backend-test test frontend-dev backend-dev prod prod-all frontend-prod

DIRTY_TREE := $(shell git diff-index --quiet HEAD -- || echo '+dirty')
COMMIT     := $(addsuffix $(DIRTY_TREE),$(shell git rev-parse --short HEAD))
VERSION    := 1.0.0-rc1+$(COMMIT)

# Coverage output file (placed in build/)
COVERAGE_FILE := build/backend-coverage.out

BUILD_FLAGS := -tags prod -ldflags "-X github.com/cactusdynamics/wesplot.Version=$(VERSION)"

clean:
	rm -rf build webui frontend/dist

backend-lint:
	go vet ./...
	gopls check -severity=info $(shell find . -name '*.go')

lint: backend-lint

backend-test:
	@mkdir -p build
	COVERAGE_FILE=$(COVERAGE_FILE) COVERAGE=$(COVERAGE) scripts/run-backend-test.sh

test: backend-test

backend-dev:
	# Not the best for now but whatever
	python3 scripts/fake_data.py | go run cmd/wesplot/main.go

frontend-dev:
	cd frontend && npm run dev -- --host 0.0.0.0 --port 5273

frontend-prod:
	cd frontend && npm run build
	rm -rf webui
	cp -ar frontend/dist webui

prod: frontend-prod
	CGO_ENABLED=0 go build $(BUILD_FLAGS) -o build/wesplot ./cmd/wesplot

prod-all:
	@mkdir -p build
	export GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 && go build $(BUILD_FLAGS) -o build/wesplot-$$GOOS-$$GOARCH-$(VERSION) ./cmd/wesplot
	export GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 && go build $(BUILD_FLAGS) -o build/wesplot-$$GOOS-$$GOARCH-$(VERSION) ./cmd/wesplot
	export GOOS=linux GOARCH=amd64 CGO_ENABLED=0 && go build $(BUILD_FLAGS) -o build/wesplot-$$GOOS-$$GOARCH-$(VERSION) ./cmd/wesplot
	export GOOS=linux GOARCH=arm64 CGO_ENABLED=0 && go build $(BUILD_FLAGS) -o build/wesplot-$$GOOS-$$GOARCH-$(VERSION) ./cmd/wesplot
	export GOOS=windows GOARCH=amd64 CGO_ENABLED=0 && go build $(BUILD_FLAGS) -o build/wesplot-$$GOOS-$$GOARCH-$(VERSION) ./cmd/wesplot
	cd build && sha256sum * | tee sha256sums
	ls -lh build
