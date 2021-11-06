# linter vet
check: lint2 lint1 vet
# golangci-lint (must be installed and setup)
lint1:
	golangci-lint run ./...
lint2:
	#go get -u golang.org/x/lint/golint
	#/Users/user/go/bin/
	golint $$(go list ./... | grep -v native)
# exe
build:
	@go build ./cmd/web-link/main.go
# test filter
test:
	@go test -v ./...
integration_test:
	@go test -v --tags=integration ./...
# vet
vet:
	go vet $$(go list ./... | grep -v native)

hooks:
	pre-commit run --all-files

test_cover:
	go test --tags=integration -v -coverprofile cover.out ./...
	go tool cover -html=cover.out -o cover.html
	open cover.html

.PHONY: test check build vet lint1 lint2 hooks test_cover integration_test
