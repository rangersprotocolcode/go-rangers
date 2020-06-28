.PHONY: all
all:
	export GOPROXY="http://goproxy.cn"
	export GO111MODULE=on
	go mod vendor
	go build -v -mod vendor -o rocket-node src/gx/main.go

.PHONY: clean
clean:
	go clean -x -cache
