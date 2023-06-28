// Copyright 2020 The RangersProtocol Authors
// This file is part of the RangersProtocol library.
//
// The RangersProtocol library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The RangersProtocol library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the RangersProtocol library. If not, see <http://www.gnu.org/licenses/>.

package vm

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/middleware/log"
	"fmt"
	"math/big"
	"os"
	"strconv"
	"testing"
)

func TestMockCall(t *testing.T) {
	defer func() {
		os.RemoveAll("storage0")
		os.RemoveAll("logs")
		os.RemoveAll("1.ini")
	}()
	mockInit()
	config := new(testConfig)
	setDefaults(config)
	defer log.Close()

	config.Origin = common.HexToAddress("0xCA8E4c934CF34e22b578ECe48c657f02B1053367")
	config.GasLimit = 3000000
	config.GasPrice = big.NewInt(1)

	contractCodeBytes := common.Hex2Bytes("608060405234801561001057600080fd5b50604051610e84380380610e848339818101604052602081101561003357600080fd5b8101908080519060200190929190505050336000806101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff1602179055506000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16600073ffffffffffffffffffffffffffffffffffffffff167f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e060405160405180910390a3600073ffffffffffffffffffffffffffffffffffffffff168173ffffffffffffffffffffffffffffffffffffffff1614156101a3576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004018080602001828103825260208152602001807f636f6e7374727563746f7220616464722063616e206e6f74206265207a65726f81525060200191505060405180910390fd5b80600160006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff1602179055507fa205958627e1d904a21bb9da37a75abcd93b5e27c98513b93c667420c28265ad600160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff16604051808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390a150610c0b806102796000396000f3fe6080604052600436106100705760003560e01c806388ef2e731161004e57806388ef2e73146101cd5780638da5cb5b1461021e5780638f32d59b14610275578063f2fde38b146102a457610070565b80630cb3a7e7146100755780635afb526b1461015f578063715018a6146101b6575b600080fd5b34801561008157600080fd5b506101456004803603604081101561009857600080fd5b8101908080359060200190929190803590602001906401000000008111156100bf57600080fd5b8201836020820111156100d157600080fd5b803590602001918460018302840111640100000000831117156100f357600080fd5b91908080601f016020809104026020016040519081016040528093929190818152602001838380828437600081840152601f19601f8201169050808301925050505050505091929192905050506102f5565b604051808215151515815260200191505060405180910390f35b34801561016b57600080fd5b506101746103e8565b604051808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390f35b3480156101c257600080fd5b506101cb610412565b005b3480156101d957600080fd5b5061021c600480360360208110156101f057600080fd5b81019080803573ffffffffffffffffffffffffffffffffffffffff16906020019092919050505061054b565b005b34801561022a57600080fd5b50610233610714565b604051808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390f35b34801561028157600080fd5b5061028a61073d565b604051808215151515815260200191505060405180910390f35b3480156102b057600080fd5b506102f3600480360360208110156102c757600080fd5b81019080803573ffffffffffffffffffffffffffffffffffffffff169060200190929190505050610794565b005b6000604182511461036e576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040180806020018281038252600f8152602001807f7369676e73206d757374203d203635000000000000000000000000000000000081525060200191505060405180910390fd5b600061037a848461081a565b9050600160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168173ffffffffffffffffffffffffffffffffffffffff1614156103dc5760019150506103e2565b60009150505b92915050565b6000600160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff16905090565b61041a61073d565b61048c576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004018080602001828103825260208152602001807f4f776e61626c653a2063616c6c6572206973206e6f7420746865206f776e657281525060200191505060405180910390fd5b600073ffffffffffffffffffffffffffffffffffffffff166000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff167f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e060405160405180910390a360008060006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff160217905550565b61055361073d565b6105c5576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004018080602001828103825260208152602001807f4f776e61626c653a2063616c6c6572206973206e6f7420746865206f776e657281525060200191505060405180910390fd5b600073ffffffffffffffffffffffffffffffffffffffff168173ffffffffffffffffffffffffffffffffffffffff16141561064b576040517f08c379a0000000000000000000000000000000000000000000000000000000008152600401808060200182810382526024815260200180610b6d6024913960400191505060405180910390fd5b80600160006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff1602179055507fa205958627e1d904a21bb9da37a75abcd93b5e27c98513b93c667420c28265ad600160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff16604051808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390a150565b60008060009054906101000a900473ffffffffffffffffffffffffffffffffffffffff16905090565b60008060009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff1614905090565b61079c61073d565b61080e576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004018080602001828103825260208152602001807f4f776e61626c653a2063616c6c6572206973206e6f7420746865206f776e657281525060200191505060405180910390fd5b6108178161088b565b50565b60008061083261082d84600060206109cf565b610a83565b9050600061084a610845856020806109cf565b610a83565b9050600061085b85604060016109cf565b60008151811061086757fe5b602001015160f81c60f81b905061088086848484610a91565b935050505092915050565b600073ffffffffffffffffffffffffffffffffffffffff168173ffffffffffffffffffffffffffffffffffffffff161415610911576040517f08c379a0000000000000000000000000000000000000000000000000000000008152600401808060200182810382526046815260200180610b916046913960600191505060405180910390fd5b8073ffffffffffffffffffffffffffffffffffffffff166000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff167f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e060405160405180910390a3806000806101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff16021790555050565b606080826040519080825280601f01601f191660200182016040528015610a055781602001600182028038833980820191505090505b50905060008090505b83811015610a77578585820181518110610a2457fe5b602001015160f81c60f81b828281518110610a3b57fe5b60200101907effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff1916908160001a9053508080600101915050610a0e565b50809150509392505050565b600060208201519050919050565b6000808260f81c905060008360f81c60ff161480610ab5575060018360f81c60ff16145b15610ac457601b8360f81c0190505b7f7fffffffffffffffffffffffffffffff5d576e7357a4501ddfe92f46681b20a08460001c1115610af9576000915050610b64565b60018682878760405160008152602001604052604051808581526020018460ff1660ff1681526020018381526020018281526020019450505050506020604051602081039080840390855afa158015610b56573d6000803e3d6000fd5b505050602060405103519150505b94935050505056fe757064617465436865636b4164647220616464722063616e206e6f74206265207a65726f4f776e61626c653a205f7472616e736665724f776e6572736869702063616e206e6f74207472616e73666572206f776e65727368697020746f207a65726f2061646472657373a265627a7a723158204c1879aab98d2291eb4f4531ef51904414d6a2034f8845f4e2b5f667430ab53264736f6c63430005110032000000000000000000000000eeddb2174a69158c85439ea8f20ae19d11f2e5db")
	createResult, multiSign, createLeftGas, createErr := mockCreate(contractCodeBytes, config)
	fmt.Printf("New create contract address:%s\n", multiSign.GetHexString())
	fmt.Printf("New create contract createResult:%v,%d\n", createResult, len(createResult))
	fmt.Printf("New create contract costGas:%v,createErr:%v\n", config.GasLimit-createLeftGas, createErr)
	fmt.Println()

	contractCodeBytes = common.Hex2Bytes("6080604052600060045534801561001557600080fd5b506129a2806100256000396000f3fe60806040526004361061007b5760003560e01c806374ff50d61161004e57806374ff50d6146104a457806385166594146104d5578063cd138f1a14610575578063d3ed7c76146106155761007b565b8063046b2908146101c45780630bceb974146101db5780632870b7a4146102a657806335c7071f146103a5575b600034116100d0576040805162461bcd60e51b815260206004820152601e60248201527f66616c6c6261636b2072657175697265206d73672e76616c7565203e20300000604482015290519081900360640190fd5b6002546040516001600160a01b03909116903480156108fc02916000818181858888f19350505050158015610109573d6000803e3d6000fd5b5060408051600080825233928201839052346060838101829052608060208501818152825182870152825192967f6d976b1592bf05d5764ab2f7bf4f8a701dcbb17110f083a80ae407a34c6705e29688959194919390929160a0840191908083838b5b8381101561018457818101518382015260200161016c565b50505050905090810190601f1680156101b15780820380516001836020036101000a031916815260200191505b509550505050505060405180910390a150005b3480156101d057600080fd5b506101d9610693565b005b3480156101e757600080fd5b506101d9600480360360808110156101fe57600080fd5b810190602081018135600160201b81111561021857600080fd5b82018360208201111561022a57600080fd5b803590602001918460018302840111600160201b8311171561024b57600080fd5b91908080601f016020809104026020016040519081016040528093929190818152602001838380828437600092019190915250929550506001600160a01b038335811694506020840135811693604001351691506106ca9050565b3480156102b257600080fd5b506101d9600480360360e08110156102c957600080fd5b8135916001600160a01b0360208201351691810190606081016040820135600160201b8111156102f857600080fd5b82018360208201111561030a57600080fd5b803590602001918460018302840111600160201b8311171561032b57600080fd5b919390926001600160a01b0383358116936020810135909116926040820135929091608081019060600135600160201b81111561036757600080fd5b82018360208201111561037957600080fd5b803590602001918460018302840111600160201b8311171561039a57600080fd5b5090925090506109a6565b3480156103b157600080fd5b506101d9600480360360e08110156103c857600080fd5b8135916001600160a01b0360208201351691810190606081016040820135600160201b8111156103f757600080fd5b82018360208201111561040957600080fd5b803590602001918460018302840111600160201b8311171561042a57600080fd5b919390926001600160a01b0383358116936020810135909116926040820135929091608081019060600135600160201b81111561046657600080fd5b82018360208201111561047857600080fd5b803590602001918460018302840111600160201b8311171561049957600080fd5b509092509050611014565b3480156104b057600080fd5b506104b961161c565b604080516001600160a01b039092168252519081900360200190f35b6101d9600480360360c08110156104eb57600080fd5b6001600160a01b038235811692602081013590911691810190606081016040820135600160201b81111561051e57600080fd5b82018360208201111561053057600080fd5b803590602001918460018302840111600160201b8311171561055157600080fd5b91935091506001600160a01b0381358116916020810135909116906040013561162b565b6101d9600480360360c081101561058b57600080fd5b6001600160a01b038235811692602081013590911691810190606081016040820135600160201b8111156105be57600080fd5b8201836020820111156105d057600080fd5b803590602001918460018302840111600160201b831117156105f157600080fd5b91935091506001600160a01b03813581169160208101359091169060400135611c32565b6101d96004803603606081101561062b57600080fd5b6001600160a01b038235169190810190604081016020820135600160201b81111561065557600080fd5b82018360208201111561066757600080fd5b803590602001918460018302840111600160201b8311171561068857600080fd5b919350915035612196565b60045460408051918252517f8a75e65e6a8c5a0bc5c906bb7fc1f38cdbdfc0adc92e41b43b3ce83463a8a8c09181900360200190a1565b6000845111610715576040805162461bcd60e51b8152602060048201526012602482015271696e6974206d75737420686173206e616d6560701b604482015290519081900360640190fd5b6001600160a01b038316610770576040805162461bcd60e51b815260206004820152601a60248201527f696e6974205f616464722063616e206e6f74206265207a65726f000000000000604482015290519081900360640190fd5b6001600160a01b0382166107cb576040805162461bcd60e51b815260206004820152601c60248201527f696e6974205f7365747465722063616e206e6f74206265207a65726f00000000604482015290519081900360640190fd5b6001600160a01b038116610826576040805162461bcd60e51b815260206004820152601d60248201527f696e6974205f666565616464722063616e206e6f74206265207a65726f000000604482015290519081900360640190fd5b6003546001600160a01b031615610896576001546001600160a01b03163314610896576040805162461bcd60e51b815260206004820152601760248201527f696e6974206e6f74207365747465722063616c6c696e67000000000000000000604482015290519081900360640190fd5b83516108a9906000906020870190612570565b50600380546001600160a01b038086166001600160a01b031992831681179093556001805486831690841681179091556002805492861692909316821790925560408051602080820195909552908101929092526060820152608080825286519082015285517f42a8653e4d61794143c4316220f8ddcc01cb1d00c05a63600d702766e651cccb92879287928792879291829160a08301919088019080838360005b8381101561096357818101518382015260200161094b565b50505050905090810190601f1680156109905780820380516001836020036101000a031916815260200191505b509550505050505060405180910390a150505050565b88600454600101146109ff576040805162461bcd60e51b815260206004820152601a60248201527f5769746864726177457263373231206e6f6e6365206572726f72000000000000604482015290519081900360640190fd5b6001600160a01b038816610a445760405162461bcd60e51b815260040180806020018281038252602c815260200180612809602c913960400191505060405180910390fd5b6001600160a01b038516610a895760405162461bcd60e51b815260040180806020018281038252602a815260200180612637602a913960400191505060405180910390fd5b85610ac55760405162461bcd60e51b81526004018080602001828103825260298152602001806127ba6029913960400191505060405180910390fd5b60006040518082805460018160011615610100020316600290048015610b225780601f10610b00576101008083540402835291820191610b22565b820191906000526020600020905b815481529060010190602001808311610b0e575b505091505060405180910390208787604051808383808284378083019250505092505050604051809103902014610ba0576040805162461bcd60e51b815260206004820152601f60248201527f5769746864726177457263373231205f66726f6d636861696e206572726f7200604482015290519081900360640190fd5b6001600160a01b038416610be55760405162461bcd60e51b81526004018080602001828103825260248152602001806126886024913960400191505060405180910390fd5b60418114610c245760405162461bcd60e51b81526004018080602001828103825260268152602001806128826026913960400191505060405180910390fd5b60608989898989898960405160200180888152602001876001600160a01b03166001600160a01b031660601b8152601401868680828437606096871b6bffffffffffffffffffffffff19908116919093019081529490951b166014840152506028808301919091526040805180840390920182526048830180825282516020840120600354630cb3a7e760e01b909252604c8501818152606c8601938452608c86018d9052939a5098506001600160a01b03169650630cb3a7e795508794508a9350899260ac01848480828437600083820152604051601f909101601f1916909201965060209550909350505081840390508186803b158015610d2657600080fd5b505afa158015610d3a573d6000803e3d6000fd5b505050506040513d6020811015610d5057600080fd5b5051610d5d575050611009565b600480546001019081905560408051918252517f8a75e65e6a8c5a0bc5c906bb7fc1f38cdbdfc0adc92e41b43b3ce83463a8a8c09181900360200190a1306001600160a01b03168a6001600160a01b0316636352211e876040518263ffffffff1660e01b81526004018082815260200191505060206040518083038186803b158015610de857600080fd5b505afa158015610dfc573d6000803e3d6000fd5b505050506040513d6020811015610e1257600080fd5b50516001600160a01b03161415610f5057604080516323b872dd60e01b81523060048201526001600160a01b038881166024830152604482018890529151918c16916323b872dd9160648082019260009290919082900301818387803b158015610e7b57600080fd5b505af1158015610e8f573d6000803e3d6000fd5b505050507ff19f37313cf850d6f04a0690ebf3adaf743ce95a2fb9c2c6443fa8bce8ece2908a8a8a8a8a8a60405180876001600160a01b03166001600160a01b0316815260200180602001856001600160a01b03166001600160a01b03168152602001846001600160a01b03166001600160a01b031681526020018381526020018281038252878782818152602001925080828437600083820152604051601f909101601f1916909201829003995090975050505050505050a15050611009565b7fc43b00ac24a4afa3154399bfa1522f6f082712bb34ca394fb3172e4d4b3b2db68a8a8a8a8a8a60405180876001600160a01b03166001600160a01b0316815260200180602001856001600160a01b03166001600160a01b03168152602001846001600160a01b03166001600160a01b031681526020018381526020018281038252878782818152602001925080828437600083820152604051601f909101601f1916909201829003995090975050505050505050a150505b505050505050505050565b886004546001011461106d576040805162461bcd60e51b815260206004820152601960248201527f57697468647261774572633230206e6f6e6365206572726f7200000000000000604482015290519081900360640190fd5b6001600160a01b0388166110b25760405162461bcd60e51b815260040180806020018281038252602b8152602001806126ac602b913960400191505060405180910390fd5b6001600160a01b0385166110f75760405162461bcd60e51b815260040180806020018281038252602981526020018061276c6029913960400191505060405180910390fd5b856111335760405162461bcd60e51b81526004018080602001828103825260288152602001806128cc6028913960400191505060405180910390fd5b600060405180828054600181600116156101000203166002900480156111905780601f1061116e576101008083540402835291820191611190565b820191906000526020600020905b81548152906001019060200180831161117c575b50509150506040518091039020878760405180838380828437808301925050509250505060405180910390201461120e576040805162461bcd60e51b815260206004820152601e60248201527f57697468647261774572633230205f66726f6d636861696e206572726f720000604482015290519081900360640190fd5b6001600160a01b0384166112535760405162461bcd60e51b81526004018080602001828103825260238152602001806127266023913960400191505060405180910390fd5b604181146112925760405162461bcd60e51b81526004018080602001828103825260268152602001806127006026913960400191505060405180910390fd5b60608989898989898960405160200180888152602001876001600160a01b03166001600160a01b031660601b8152601401868680828437606096871b6bffffffffffffffffffffffff19908116919093019081529490951b166014840152506028808301919091526040805180840390920182526048830180825282516020840120600354630cb3a7e760e01b909252604c8501818152606c8601938452608c86018d9052939a5098506001600160a01b03169650630cb3a7e795508794508a9350899260ac01848480828437600083820152604051601f909101601f1916909201965060209550909350505081840390508186803b15801561139457600080fd5b505afa1580156113a8573d6000803e3d6000fd5b505050506040513d60208110156113be57600080fd5b50516113cb575050611009565b600480546001019081905560408051918252517f8a75e65e6a8c5a0bc5c906bb7fc1f38cdbdfc0adc92e41b43b3ce83463a8a8c09181900360200190a1604080516370a0823160e01b8152306004820152905186916001600160a01b038d16916370a0823191602480820192602092909190829003018186803b15801561145157600080fd5b505afa158015611465573d6000803e3d6000fd5b505050506040513d602081101561147b57600080fd5b5051106115595761149c6001600160a01b038b16878763ffffffff6122fb16565b7f4504ae79b970114e53fe85acff5089a540b0d9be3cdddbcd67c14160f9bca6c08a8a8a8a8a8a60405180876001600160a01b03166001600160a01b0316815260200180602001856001600160a01b03166001600160a01b03168152602001846001600160a01b03166001600160a01b031681526020018381526020018281038252878782818152602001925080828437600083820152604051601f909101601f1916909201829003995090975050505050505050a15050611009565b7fbc3df4d7f7795a1659d727b7ccd1c240eeb431773a685c61eb604f12267305cc8a8a8a8a8a8a60405180876001600160a01b03166001600160a01b0316815260200180602001856001600160a01b03166001600160a01b03168152602001846001600160a01b03166001600160a01b031681526020018381526020018281038252878782818152602001925080828437600083820152604051601f909101601f1916909201829003995090975050505050505050a15050505050505050505050565b6002546001600160a01b031681565b6001600160a01b0387166116705760405162461bcd60e51b815260040180806020018281038252602b81526020018061260c602b913960400191505060405180910390fd5b6001600160a01b0386166116b55760405162461bcd60e51b81526004018080602001828103825260298152602001806126d76029913960400191505060405180910390fd5b836116f15760405162461bcd60e51b81526004018080602001828103825260268152602001806127e36026913960400191505060405180910390fd5b6001600160a01b0383166117365760405162461bcd60e51b81526004018080602001828103825260278152602001806126616027913960400191505060405180910390fd5b6001600160a01b03821661177b5760405162461bcd60e51b815260040180806020018281038252602581526020018061285d6025913960400191505060405180910390fd5b600034116117d0576040805162461bcd60e51b815260206004820152601a60248201527f446f43726f7373457263373231206d7573742068617320666565000000000000604482015290519081900360640190fd5b336001600160a01b0384161461182d576040805162461bcd60e51b815260206004820152601d60248201527f446f43726f73734572633732312077726f6e67205f66726f6d61646472000000604482015290519081900360640190fd5b6002546040516001600160a01b03909116903480156108fc02916000818181858888f19350505050158015611866573d6000803e3d6000fd5b50826001600160a01b0316876001600160a01b0316636352211e836040518263ffffffff1660e01b81526004018082815260200191505060206040518083038186803b1580156118b557600080fd5b505afa1580156118c9573d6000803e3d6000fd5b505050506040513d60208110156118df57600080fd5b50516001600160a01b03161480156119fe5750306001600160a01b0316876001600160a01b031663081812fc836040518263ffffffff1660e01b81526004018082815260200191505060206040518083038186803b15801561194057600080fd5b505afa158015611954573d6000803e3d6000fd5b505050506040513d602081101561196a57600080fd5b50516001600160a01b031614806119fe57506040805163e985e9c560e01b81526001600160a01b03858116600483015230602483015291519189169163e985e9c591604480820192602092909190829003018186803b1580156119cc57600080fd5b505afa1580156119e0573d6000803e3d6000fd5b505050506040513d60208110156119f657600080fd5b505115156001145b15611b5057604080516323b872dd60e01b81526001600160a01b038581166004830152306024830152604482018490529151918916916323b872dd9160648082019260009290919082900301818387803b158015611a5b57600080fd5b505af1158015611a6f573d6000803e3d6000fd5b505050507fd304ed7a7c7ca6abe31a5f552b8bbcb9bb22f3708cdfc7258a922db04197842b348888888888888860405180898152602001886001600160a01b03166001600160a01b03168152602001876001600160a01b03166001600160a01b0316815260200180602001856001600160a01b03166001600160a01b03168152602001846001600160a01b03166001600160a01b031681526020018381526020018281038252878782818152602001925080828437600083820152604051601f909101601f19169092018290039b50909950505050505050505050a1611c29565b7faa37f29a916d6e76bdd96037490123d26b69c1f01931ca5c808f02d7bf06d9bf348888888888888860405180898152602001886001600160a01b03166001600160a01b03168152602001876001600160a01b03166001600160a01b0316815260200180602001856001600160a01b03166001600160a01b03168152602001846001600160a01b03166001600160a01b031681526020018381526020018281038252878782818152602001925080828437600083820152604051601f909101601f19169092018290039b50909950505050505050505050a15b50505050505050565b6001600160a01b038716611c775760405162461bcd60e51b815260040180806020018281038252602a8152602001806128f4602a913960400191505060405180910390fd5b6001600160a01b038616611cbc5760405162461bcd60e51b81526004018080602001828103825260288152602001806128356028913960400191505060405180910390fd5b83611cf85760405162461bcd60e51b81526004018080602001828103825260258152602001806127956025913960400191505060405180910390fd5b6001600160a01b038316611d3d5760405162461bcd60e51b81526004018080602001828103825260268152602001806129486026913960400191505060405180910390fd5b6001600160a01b038216611d825760405162461bcd60e51b81526004018080602001828103825260248152602001806128a86024913960400191505060405180910390fd5b60008111611dc15760405162461bcd60e51b81526004018080602001828103825260238152602001806127496023913960400191505060405180910390fd5b60003411611e16576040805162461bcd60e51b815260206004820152601960248201527f446f43726f73734572633230206d757374206861732066656500000000000000604482015290519081900360640190fd5b336001600160a01b03841614611e73576040805162461bcd60e51b815260206004820152601c60248201527f446f43726f737345726332302077726f6e67205f66726f6d6164647200000000604482015290519081900360640190fd5b6002546040516001600160a01b03909116903480156108fc02916000818181858888f19350505050158015611eac573d6000803e3d6000fd5b5080876001600160a01b03166370a08231856040518263ffffffff1660e01b815260040180826001600160a01b03166001600160a01b0316815260200191505060206040518083038186803b158015611f0457600080fd5b505afa158015611f18573d6000803e3d6000fd5b505050506040513d6020811015611f2e57600080fd5b505110801590611fb8575060408051636eb1769f60e11b81526001600160a01b038581166004830152306024830152915183928a169163dd62ed3e916044808301926020929190829003018186803b158015611f8957600080fd5b505afa158015611f9d573d6000803e3d6000fd5b505050506040513d6020811015611fb357600080fd5b505110155b156120b557611fd86001600160a01b03881684308463ffffffff61235216565b7fad43c264a23265278792c8a50133a738a0762c733d8ee602fb643eead18c2a85348888888888888860405180898152602001886001600160a01b03166001600160a01b03168152602001876001600160a01b03166001600160a01b0316815260200180602001856001600160a01b03166001600160a01b03168152602001846001600160a01b03166001600160a01b031681526020018381526020018281038252878782818152602001925080828437600083820152604051601f909101601f19169092018290039b50909950505050505050505050a1611c29565b7f213592da2bfda5f5ccc1081e9f1c63295f9c3dab82f3610229761722b490e439348888888888888860405180898152602001886001600160a01b03166001600160a01b03168152602001876001600160a01b03166001600160a01b0316815260200180602001856001600160a01b03166001600160a01b03168152602001846001600160a01b03166001600160a01b031681526020018381526020018281038252878782818152602001925080828437600083820152604051601f909101601f19169092018290039b50909950505050505050505050a150505050505050565b6001600160a01b038416612237578034146121f8576040805162461bcd60e51b815260206004820152601c60248201527f7370656564557020696e73756666696369656e7420666565206e756d00000000604482015290519081900360640190fd5b6002546040516001600160a01b03909116903480156108fc02916000818181858888f19350505050158015612231573d6000803e3d6000fd5b50612259565b600254612259906001600160a01b03868116913391168463ffffffff61235216565b7f6d976b1592bf05d5764ab2f7bf4f8a701dcbb17110f083a80ae407a34c6705e2848484338560405180866001600160a01b03166001600160a01b0316815260200180602001846001600160a01b03166001600160a01b031681526020018381526020018281038252868682818152602001925080828437600083820152604051601f909101601f19169092018290039850909650505050505050a150505050565b604080516001600160a01b038416602482015260448082018490528251808303909101815260649091019091526020810180516001600160e01b031663a9059cbb60e01b17905261234d9084906123b2565b505050565b604080516001600160a01b0385811660248301528416604482015260648082018490528251808303909101815260849091019091526020810180516001600160e01b03166323b872dd60e01b1790526123ac9085906123b2565b50505050565b6123c4826001600160a01b031661256a565b612415576040805162461bcd60e51b815260206004820152601f60248201527f5361666545524332303a2063616c6c20746f206e6f6e2d636f6e747261637400604482015290519081900360640190fd5b60006060836001600160a01b0316836040518082805190602001908083835b602083106124535780518252601f199092019160209182019101612434565b6001836020036101000a0380198251168184511680821785525050505050509050019150506000604051808303816000865af19150503d80600081146124b5576040519150601f19603f3d011682016040523d82523d6000602084013e6124ba565b606091505b509150915081612511576040805162461bcd60e51b815260206004820181905260248201527f5361666545524332303a206c6f772d6c6576656c2063616c6c206661696c6564604482015290519081900360640190fd5b8051156123ac5780806020019051602081101561252d57600080fd5b50516123ac5760405162461bcd60e51b815260040180806020018281038252602a81526020018061291e602a913960400191505060405180910390fd5b3b151590565b828054600181600116156101000203166002900490600052602060002090601f016020900481019282601f106125b157805160ff19168380011785556125de565b828001600101855582156125de579182015b828111156125de5782518255916020019190600101906125c3565b506125ea9291506125ee565b5090565b61260891905b808211156125ea57600081556001016125f4565b9056fe446f43726f7373457263373231205f66726f6d636f6e74726163742063616e206e6f74206265207a65726f5769746864726177457263373231205f746f636f6e74726163742063616e206e6f74206265207a65726f446f43726f7373457263373231205f66726f6d616464722063616e206e6f74206265207a65726f5769746864726177457263373231205f616464722063616e206e6f74206265207a65726f57697468647261774572633230205f66726f6d636f6e74726163742063616e206e6f74206265207a65726f446f43726f7373457263373231205f746f636f6e74726163742063616e206e6f74206265207a65726f57697468647261774572633230205f7369676e73206c656e677468206d75737420626520363557697468647261774572633230205f616464722063616e206e6f74206265207a65726f446f43726f7373457263323020616d6f756e742063616e206e6f74206265207a65726f57697468647261774572633230205f746f636f6e74726163742063616e206e6f74206265207a65726f446f43726f73734572633230205f746f436861696e2063616e206e6f74206265206e756c6c5769746864726177457263373231205f66726f6d636861696e2063616e206e6f74206265206e756c6c446f43726f7373457263373231205f746f436861696e2063616e206e6f74206265206e756c6c5769746864726177457263373231205f66726f6d636f6e74726163742063616e206e6f74206265207a65726f446f43726f73734572633230205f746f636f6e74726163742063616e206e6f74206265207a65726f446f43726f7373457263373231205f746f616464722063616e206e6f74206265207a65726f5769746864726177457263373231207369676e73206c656e677468206d757374206265203635446f43726f73734572633230205f746f616464722063616e206e6f74206265207a65726f57697468647261774572633230205f66726f6d636861696e2063616e206e6f74206265206e756c6c446f43726f73734572633230205f61646472636f6e74726163742063616e206e6f74206265207a65726f5361666545524332303a204552433230206f7065726174696f6e20646964206e6f742073756363656564446f43726f73734572633230205f66726f6d616464722063616e206e6f74206265207a65726fa265627a7a723158201e838fd5f305e1306f34b1ef378a0166040281ce11ac7b3911c3570417791db464736f6c63430005110032")
	createResult, cross, createLeftGas, createErr := mockCreate(contractCodeBytes, config)
	fmt.Printf("New create contract address:%s\n", cross.GetHexString())
	fmt.Printf("New create contract createResult:%v,%d\n", createResult, len(createResult))
	fmt.Printf("New create contract costGas:%v,createErr:%v\n", config.GasLimit-createLeftGas, createErr)
	fmt.Println()

	contractCodeBytes = common.Hex2Bytes("608060405234801561001057600080fd5b5060405161092c38038061092c8339818101604052602081101561003357600080fd5b8101908080519060200190929190505050808060405180806108ce60239139602301905060405180910390207f7050c9e0f4ca769c69bd3a8ef740bc37934f8e2c036e5a723fd8ee048ed3f8c360001b1461008a57fe5b6100998161011160201b60201c565b5060405180807f6f72672e7a657070656c696e6f732e70726f78792e61646d696e000000000000815250601a01905060405180910390207f10d6a54a4754c8869d6886b5f5d7fbfa5b4522237ea5c60d11bc4e7a1ff9390b60001b146100fb57fe5b61010a336101a860201b60201c565b50506101ea565b610124816101d760201b6105ea1760201c565b610179576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040180806020018281038252603b8152602001806108f1603b913960400191505060405180910390fd5b60007f7050c9e0f4ca769c69bd3a8ef740bc37934f8e2c036e5a723fd8ee048ed3f8c360001b90508181555050565b60007f10d6a54a4754c8869d6886b5f5d7fbfa5b4522237ea5c60d11bc4e7a1ff9390b60001b90508181555050565b600080823b905060008111915050919050565b6106d5806101f96000396000f3fe60806040526004361061003f5760003560e01c80633659cfe6146100495780635c60da1b1461009a5780638f283970146100f1578063f851a44014610142575b610047610199565b005b34801561005557600080fd5b506100986004803603602081101561006c57600080fd5b81019080803573ffffffffffffffffffffffffffffffffffffffff1690602001909291905050506101b3565b005b3480156100a657600080fd5b506100af610208565b604051808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390f35b3480156100fd57600080fd5b506101406004803603602081101561011457600080fd5b81019080803573ffffffffffffffffffffffffffffffffffffffff169060200190929190505050610217565b005b34801561014e57600080fd5b50610157610390565b604051808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390f35b6101a161039f565b6101b16101ac610435565b610466565b565b6101bb61048c565b73ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff1614156101fc576101f7816104bd565b610205565b610204610199565b5b50565b6000610212610435565b905090565b61021f61048c565b73ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff16141561038457600073ffffffffffffffffffffffffffffffffffffffff168173ffffffffffffffffffffffffffffffffffffffff1614156102d8576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004018080602001828103825260368152602001806106306036913960400191505060405180910390fd5b7f7e644d79422f17c01e4894b5f4f588d331ebfa28653d42ae832dc59e38c9798f61030161048c565b82604051808373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020018273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019250505060405180910390a161037f8161052c565b61038d565b61038c610199565b5b50565b600061039a61048c565b905090565b6103a761048c565b73ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff16141561042b576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004018080602001828103825260328152602001806105fe6032913960400191505060405180910390fd5b61043361055b565b565b6000807f7050c9e0f4ca769c69bd3a8ef740bc37934f8e2c036e5a723fd8ee048ed3f8c360001b9050805491505090565b60005136600082376000803683855af43d6000833e8060008114610488573d83f35b3d83fd5b6000807f10d6a54a4754c8869d6886b5f5d7fbfa5b4522237ea5c60d11bc4e7a1ff9390b60001b9050805491505090565b6104c68161055d565b7fbc7cd75a20ee27fd9adebab32041f755214dbc6bffa90cc0225b39da2e5c2d3b81604051808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390a150565b60007f10d6a54a4754c8869d6886b5f5d7fbfa5b4522237ea5c60d11bc4e7a1ff9390b60001b90508181555050565b565b610566816105ea565b6105bb576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040180806020018281038252603b815260200180610666603b913960400191505060405180910390fd5b60007f7050c9e0f4ca769c69bd3a8ef740bc37934f8e2c036e5a723fd8ee048ed3f8c360001b90508181555050565b600080823b90506000811191505091905056fe43616e6e6f742063616c6c2066616c6c6261636b2066756e6374696f6e2066726f6d207468652070726f78792061646d696e43616e6e6f74206368616e6765207468652061646d696e206f6620612070726f787920746f20746865207a65726f206164647265737343616e6e6f742073657420612070726f787920696d706c656d656e746174696f6e20746f2061206e6f6e2d636f6e74726163742061646472657373a265627a7a72315820f402c332f24e5ad4605bc293203bbc4fd0d24ab8c571ae12814d7a9d7eb9410f64736f6c634300051100326f72672e7a657070656c696e6f732e70726f78792e696d706c656d656e746174696f6e43616e6e6f742073657420612070726f787920696d706c656d656e746174696f6e20746f2061206e6f6e2d636f6e74726163742061646472657373000000000000000000000000ffcef323281452f4e438f2c305789cd20d8eeff6")
	createResult, proxy, createLeftGas, createErr := mockCreate(contractCodeBytes, config)
	fmt.Printf("New create contract address:%s\n", proxy.GetHexString())
	fmt.Printf("New create contract createResult:%v,%d\n", createResult, len(createResult))
	fmt.Printf("New create contract costGas:%v,createErr:%v\n", config.GasLimit-createLeftGas, createErr)
	fmt.Println()

	config.Origin = common.HexToAddress("0x7edd0ef9da9cec334a7887966cc8dd71d590eeb7")
	maincontractinput := "0bceb9740000000000000000000000000000000000000000000000000000000000000080000000000000000000000000c2098a8938119a52b1f7661893c0153a6cb116d50000000000000000000000007edd0ef9da9cec334a7887966cc8dd71d590eeb70000000000000000000000007c8a141e0b9ad3a79b5084fc85a38dc7b08ea7fa00000000000000000000000000000000000000000000000000000000000000036161610000000000000000000000000000000000000000000000000000000000"
	callResult, callLeftGas, callErr := mockCall(proxy, common.FromHex(maincontractinput), config)

	fmt.Printf("callResult: %v,costGas: %d,callErr: %v\n", callResult, config.GasLimit-callLeftGas, callErr)

}

func TestMockData(t *testing.T) {
	//0000000000000000000000000000000000000000000000000000000000000003
	//6161610000000000000000000000000000000000000000000000000000000000

	//
	//
	str := "aaaaaaaaaaaaaaaaaaaa"
	length := strconv.FormatUint(uint64(len(str)), 16)
	padding := 64 - len(length)
	for i := 0; i < padding; i++ {
		length = "0" + length
	}
	fmt.Println(length)

	data := common.Bytes2Hex([]byte(str))
	padding = 64 - len(data)
	for i := 0; i < padding; i++ {
		data += "0"
	}
	fmt.Println(data)

}
