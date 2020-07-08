# Rocket Node

go implementation for RocketProtocol

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

# 2 Running from command line
### 2.1 for new nodes

if you do not have an account, RocketProtocol node will generate a new one. Check rp.ini after running the following command

#### 2.1.1 connect to testnet
```
./rocket-node miner
```

You can use browser, Chrome for example, to check your node status by url: http://0.0.0.0:8088/

#### 2.1.2 connect to mainnet
```
./rocket-node miner --env production
```

Also you can use browser, Chrome for example, to check your node status by url: http://0.0.0.0:8088/

### 2.2 for existing nodes
If you have your privatekey and address, you have to specify them in rp.ini. 

For example: 
```
[gx]
privatekey = 0x04cabf57e62b454d2bd63e56927ee04137c4a922772045ea06080c255de3776219f54ecfef798e309d0a6d0bd75e6ed2923db24e208580d77b87b9b5ee894dd241d9ae7e257a7803d760088170d1fb851d99f1039ee44506b9211fd2c77ff3e0df
miner = 0xe72c1487d6940098832ef95486164b84605a60a70d4aea58a865c20d509c42f8
```

Use config to specify your owner file. For example:
```
./rocket-node miner --config rp.ini --env production 
```

# 3 Start mining
RocketProtocol has two roles: proposer and validator. You have to apply one of the two roles.

Please contact us if you want to know the apply process

# 4 Build your own testnet locally

Check our MVP documents: [MVP documents](http://git.tuntunhz.com/tequila/rockectmvp)
