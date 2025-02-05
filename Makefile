# linter vet
check: lint2 lint1 vet
# golangci-lint (must be installed and setup)
lint1:
	golangci-lint run ./...
lint2:
	#go get -u golang.org/x/lint/golint
	/Users/user/go/bin/golint $$(go list ./... | grep -v native)
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

build_arm:
	# build image for raspberry pi arm64: (cross to arm)
	docker buildx build --platform linux/arm64 -t web-link:arm64 .
	# copy docker image  to raspberry system:
	# save it in tar archive
	docker save web-link:arm64  > weblink-arm64.tar
	# copy to pi via scp will ask for remote user passwd if not via ssh-cert
	scp weblink-arm64.tar user@192.168.1.204:/home/user
	# unpack tar to local image docker repo (on pi side)
	# docker load -i weblink-arm64.tar
	# check & run
	# docker image ls | grep arm
	# docker run -p 8000:8000 web-link:arm64
    # move container from docker to kube (after save tar then import locally to kube local store)
	# microk8s ctr image import weblink-arm64.tar

copy_kube:
	scp -r ../kube user@192.168.1.210:/home/user/


.PHONY: test check build vet lint1 lint2 hooks test_cover integration_test build_arm copy_kube
