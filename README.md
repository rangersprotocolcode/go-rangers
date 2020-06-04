# Rocket Node

go implementation for layer2

## Prerequisite

- Install Go binary release from [https://golang.org/dl/](https://golang.org/dl/).
- Ensure a C compiler is working in case of cgo dependencies.

## Build & Clean

To build this project:

```sh
make
```

To clean build:

```sh
make clean
```

## Package Management

We use go modules to manage all dependencies.

Check official documents before add or update any third party packages:

- [https://github.com/golang/go/wiki/Modules](https://github.com/golang/go/wiki/Modules)
- [https://blog.golang.org/using-go-modules](https://blog.golang.org/using-go-modules)

bash_profile
- export GOPROXY="http://goproxy.cn" 

- export GO111MODULE=on