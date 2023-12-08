# Rangers Node

go implementation for RangersProtocol

# 0 Overview

For more details about RocketProtocol please refer to the lightpaper: [lightpaper link](http://git.tuntunhz.com/tequila/opendocs/-/blob/master/0000-rocket-lightpaper/lightpaper.md)

# 1 Build an executable binary

### 1.1 prerequisite

- Install Go binary release 1.13 from

  [https://studygolang.com/dl](https://studygolang.com/dl) or

  [https://golang.org/dl/](https://golang.org/dl/)
- Ensure a C compiler is working because of the cgo dependencies

### 1.2 download code
```
git clone git@github.com:rangersprotocolcode/go-rangers.git
```

make sure you have the right account/password

### 1.3 build & c

To build this project:

```sh
make
```

To clean build:

```sh
make clean
```

Check if you have the binary file: rocket-node

### 1.4 package management

We use go modules to manage all dependencies.

Check official documents before add or update any third party packages:

- [https://github.com/golang/go/wiki/Modules](https://github.com/golang/go/wiki/Modules)
- [https://blog.golang.org/using-go-modules](https://blog.golang.org/using-go-modules)

You may need the proxy for downloading the go packages.

for example: bash_profile for linux
- export GOPROXY = "http://goproxy.cn" 
- export GO111MODULE = on
