.PHONY: frontend-dev backend-dev prod

DIRTY_TREE := $(shell git diff-index --quiet HEAD -- || echo '+dirty')
COMMIT     := $(addsuffix $(DIRTY_TREE),$(shell git rev-parse --short HEAD))
VERSION    := 0.99.0+$(COMMIT)

backend-dev:
	# Not the best for now but whatever
	python3 scripts/fake_data.py | go run cmd/main.go

frontend-dev:
	cd frontend && yarn dev --host 0.0.0.0 --port 5273

prod:
	cd frontend && yarn build
	rm -rf webui
	cp -ar frontend/dist webui
	mkdir -p build
	# TODO: Make different architecture variants as well
	go build -tags prod -o build/wesplot -ldflags "-X github.com/cactusdynamics/wesplot.Version=$(VERSION)" ./cmd
