.PHONY: all
all:
	go build -v -mod vendor -o rocket-node src/gx/main.go

.PHONY: clean
clean:
	go clean -x -cache
