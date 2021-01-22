.PHONY: test

test:
	@echo "[go test] running tests and collecting coverage metrics"
	@go test -v -tags all_tests -race -coverprofile=coverage.txt -covermode=atomic ./...

lint: lint-check
	@echo "[golangci-lint] linting sources"
	@golangci-lint run \
		-E misspell \
	  	-E golint \
	  	-E gofmt \
	  	-E unconvert \
	  	--exclude-use-default=false \
	  	./...

fmt:
	go fmt ./...

lint-check:
	@if [ -z `which golangci-lint` ]; then \
		echo "[go get] installing golangci-lint";\
		GO111MODULE=on go get -u github.com/golangci/golangci-lint/cmd/golangci-lint;\
	fi