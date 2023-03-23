.PHONY: frontend-dev

backend-dev:
	# Not the best for now but whatever
	python3 scripts/fake_data.py | go run cmd/main.go

frontend-dev:
	cd frontend && yarn dev --host 0.0.0.0 --port 5273
