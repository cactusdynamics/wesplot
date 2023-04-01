.PHONY: frontend-dev backend-dev prod prod-all frontend-prod

DIRTY_TREE := $(shell git diff-index --quiet HEAD -- || echo '+dirty')
COMMIT     := $(addsuffix $(DIRTY_TREE),$(shell git rev-parse --short HEAD))
VERSION    := 0.99.0+$(COMMIT)

BUILD_FLAGS := -tags prod -ldflags "-X github.com/cactusdynamics/wesplot.Version=$(VERSION)"

backend-dev:
	# Not the best for now but whatever
	python3 scripts/fake_data.py | go run cmd/main.go

frontend-dev:
	cd frontend && yarn dev --host 0.0.0.0 --port 5273

frontend-prod:
	cd frontend && yarn build
	rm -rf webui
	cp -ar frontend/dist webui

prod: frontend-dev
	mkdir -p build
	go build $(BUILD_FLAGS) -o build/wesplot ./cmd

prod-all: frontend-prod
	rm -rf build
	mkdir -p build
	export GOOS=darwin GOARCH=amd64 && go build $(BUILD_FLAGS) -o build/wesplot-$$GOOS-$$GOARCH ./cmd
	export GOOS=darwin GOARCH=arm64 && go build $(BUILD_FLAGS) -o build/wesplot-$$GOOS-$$GOARCH ./cmd
	export GOOS=linux GOARCH=amd64 && go build $(BUILD_FLAGS) -o build/wesplot-$$GOOS-$$GOARCH ./cmd
	export GOOS=linux GOARCH=arm64 && go build $(BUILD_FLAGS) -o build/wesplot-$$GOOS-$$GOARCH ./cmd
	export GOOS=windows GOARCH=amd64 && go build $(BUILD_FLAGS) -o build/wesplot-$$GOOS-$$GOARCH ./cmd
	cd build && sha256sum * | tee sha256sums
