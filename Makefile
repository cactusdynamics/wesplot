.PHONY: frontend-dev

backend-dev:
	# Not the best for now but whatever
	sar 1 | gawk '{ print 100-int($$NF); fflush(); }' | go run cmd/main.go

frontend-dev:
	cd frontend && yarn dev --host 0.0.0.0 --port 5273
