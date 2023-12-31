## 1 测试环境
a. 机器配置 4核8g服务器

b. 机器数量 4台，包括1个提案节点，3个验证节点

## 2 测试方式
性能测试的流程为

<image src="performance-1.png">

本次性能测试分为两种交易类型，简述如下
### 2.1 原生代币转账
我们使用Rangers的交易类型格式，构造一个向账户转账随机金额的交易。交易数据实例如下：
```json
  {"chainId":"9500","data":"","extraData":"{\"0x104949317f55C858b59A912CB1F6C501fc92Ce01\":{\"balance\":\"0.000003773620284656\"}}","hash":"0x7ca52977498bcf906bf2a6e4eaec0f291803959cde9834bba0d29baa26a45f95","nonce":0,"sign":"0x290702f85df11a340e8b4a115347204a09481f8c1e07a62bbc132319f56f3ea17c50817531ee840321d93306180c160887c389403baac50db01ddb409ed839c71c","socketRequestId":"1443920380567745849","source":"0x2c616a97d3d10e008f901b392986b1a65e0abbb7","target":"","time":"1675665727707","type":100}
```

### 2.2 ERC20合约balanceOf方法
我们使用Rangers的交易类型格式，构造一个调用ERC20合约账户的balanceOf方法，查询账户余额。交易数据实例如下
```json  
{"chainId":"9500","data":"{\"gasLimit\":\"100000000\",\"transferValue\":\"0\",\"abiData\":\"0x70a08231000000000000000000000000104949317f55c858b59a912cb1f6c501fc92ce01\",\"gasPrice\":\"1\"}","extraData":"","hash":"0xb6eac51ac6204b783e98b0b8699907a491e455194152f725d3127b3de1631640","nonce":0,"sign":"0x04ccf22001ccc11bb88dba53448e49fdc4928cc76046b1be590e84df8c7bf06728c0bbe7253b5ed169724179201faeb61f006c0b78ada15ab5e10540cd18ce451b","socketRequestId":"-1227407794021176855","source":"0x2c616a97d3d10e008f901b392986b1a65e0abbb7","target":"0x71d9cfd1b7adb1e8eb4c193ce6ffbe19b4aee0db","time":"1675667180712","type":200}
```
  
## 3 测试结果
### 3.1 原生代币转账 
  
测试网络最大tps 850。 
  
在不同的负载压力下，性能表现如下表所示：
|  tps（每秒交易数）   | cpu占用率  |
|  ----  | ----  |
| 100  | 10% |
| 200  | 12.5% |
| 300  | 16% |
| 400  | 22.5% |
| 500  | 30% |
| 600  | 40% |
| 700  | 50% |
| 800  | 70% |
| 850  | 75% |

### 3.2 ERC20合约balanceOf方法
测试网络最大tps 750。 
  
在不同的负载压力下，性能表现如下表所示：
|  tps（每秒交易数）   | cpu占用率  |
|  ----  | ----  |
| 100  | 10% |
| 200  | 12.5% |
| 300  | 20% |
| 400  | 25% |
| 500  | 35% |
| 600  | 42% |
| 700  | 50% |
| 750  | 70% | 
  
