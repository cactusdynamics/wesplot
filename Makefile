.PHONY: frontend-dev backend-dev prod prod-all frontend-prod

DIRTY_TREE := $(shell git diff-index --quiet HEAD -- || echo '+dirty')
COMMIT     := $(addsuffix $(DIRTY_TREE),$(shell git rev-parse --short HEAD))
VERSION    := 1.0.0-rc1+$(COMMIT)

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

prod: frontend-prod
	mkdir -p build
	go build $(BUILD_FLAGS) -o build/wesplot ./cmd

prod-all:
	rm -rf build
	mkdir -p build
	export GOOS=darwin GOARCH=amd64 && go build $(BUILD_FLAGS) -o build/wesplot-$$GOOS-$$GOARCH-$(VERSION) ./cmd
	export GOOS=darwin GOARCH=arm64 && go build $(BUILD_FLAGS) -o build/wesplot-$$GOOS-$$GOARCH-$(VERSION) ./cmd
	export GOOS=linux GOARCH=amd64 && go build $(BUILD_FLAGS) -o build/wesplot-$$GOOS-$$GOARCH-$(VERSION) ./cmd
	export GOOS=linux GOARCH=arm64 && go build $(BUILD_FLAGS) -o build/wesplot-$$GOOS-$$GOARCH-$(VERSION) ./cmd
	export GOOS=windows GOARCH=amd64 && go build $(BUILD_FLAGS) -o build/wesplot-$$GOOS-$$GOARCH-$(VERSION) ./cmd
	cd build && sha256sum * | tee sha256sums
	ls -lh build
