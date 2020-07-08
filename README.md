# Rocket Node

go implementation for RocketProtocol

## 0 Overview

for more details please refer to the lightpaper: [link](http://git.tuntunhz.com/tequila/opendocs/-/blob/master/0000-rocket-lightpaper/lightpaper.md)

## 1 build an executable binary

### 1.1 prerequisite

- Install Go binary release 1.13 from

  [https://studygolang.com/dl](https://studygolang.com/dl) or

  [https://golang.org/dl/](https://golang.org/dl/)
- Ensure a C compiler is working because of the cgo dependencies

### 1.2 download code
```
git clone git@git.tuntunhz.com:tequila/jojo.git
```

make sure you have the right account/password

### 1.3 build & clean

To build this project:

```sh
make
```

To clean build:

```sh
make clean
```

### 1.4 package management

We use go modules to manage all dependencies.

Check official documents before add or update any third party packages:

- [https://github.com/golang/go/wiki/Modules](https://github.com/golang/go/wiki/Modules)
- [https://blog.golang.org/using-go-modules](https://blog.golang.org/using-go-modules)

You may need the proxy for downloading the go packages

bash_profile
- export GOPROXY="http://goproxy.cn" 
- export GO111MODULE=on