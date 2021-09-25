.PHONY: all
all:
	go mod vendor
	go build -v -mod vendor -o rangers-node src/gx/main.go

.PHONY: clean
clean:
	go clean -x -cache
