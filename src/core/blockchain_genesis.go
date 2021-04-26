// Copyright 2020 The RocketProtocol Authors
// This file is part of the RocketProtocol library.
//
// The RocketProtocol library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The RocketProtocol library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the RocketProtocol library. If not, see <http://www.gnu.org/licenses/>.

package core

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/consensus/groupsig"
	"com.tuntun.rocket/node/src/consensus/vrf"
	"com.tuntun.rocket/node/src/middleware/types"
	"com.tuntun.rocket/node/src/service"
	"com.tuntun.rocket/node/src/storage/account"
	"com.tuntun.rocket/node/src/storage/trie"
	"com.tuntun.rocket/node/src/utility"
	"com.tuntun.rocket/node/src/vm"
	"math/big"
	"time"
)

const ChainDataVersion = 2

var emptyHash = common.Hash{}

var usdtContractData = "0x608060405260008060146101000a81548160ff021916908315150217905550600060035560006004553480156200003557600080fd5b5060405162002ead38038062002ead83398101806040528101908080519060200190929190805182019291906020018051820192919060200180519060200190929190505050336000806101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff160217905550836001819055508260079080519060200190620000da92919062000185565b508160089080519060200190620000f392919062000185565b508060098190555083600260008060009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020819055506000600a60146101000a81548160ff0219169083151502179055505050505062000234565b828054600181600116156101000203166002900490600052602060002090601f016020900481019282601f10620001c857805160ff1916838001178555620001f9565b82800160010185558215620001f9579182015b82811115620001f8578251825591602001919060010190620001db565b5b5090506200020891906200020c565b5090565b6200023191905b808211156200022d57600081600090555060010162000213565b5090565b90565b612c6980620002446000396000f300608060405260043610610196576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff16806306fdde031461019b5780630753c30c1461022b578063095ea7b31461026e5780630e136b19146102bb5780630ecb93c0146102ea57806318160ddd1461032d57806323b872dd1461035857806326976e3f146103c557806327e235e31461041c578063313ce56714610473578063353907141461049e5780633eaaf86b146104c95780633f4ba83a146104f457806359bf1abe1461050b5780635c658165146105665780635c975abb146105dd57806370a082311461060c5780638456cb5914610663578063893d20e81461067a5780638da5cb5b146106d157806395d89b4114610728578063a9059cbb146107b8578063c0324c7714610805578063cc872b661461083c578063db006a7514610869578063dd62ed3e14610896578063dd644f721461090d578063e47d606014610938578063e4997dc514610993578063e5b5019a146109d6578063f2fde38b14610a01578063f3bdc22814610a44575b600080fd5b3480156101a757600080fd5b506101b0610a87565b6040518080602001828103825283818151815260200191508051906020019080838360005b838110156101f05780820151818401526020810190506101d5565b50505050905090810190601f16801561021d5780820380516001836020036101000a031916815260200191505b509250505060405180910390f35b34801561023757600080fd5b5061026c600480360381019080803573ffffffffffffffffffffffffffffffffffffffff169060200190929190505050610b25565b005b34801561027a57600080fd5b506102b9600480360381019080803573ffffffffffffffffffffffffffffffffffffffff16906020019092919080359060200190929190505050610c42565b005b3480156102c757600080fd5b506102d0610d95565b604051808215151515815260200191505060405180910390f35b3480156102f657600080fd5b5061032b600480360381019080803573ffffffffffffffffffffffffffffffffffffffff169060200190929190505050610da8565b005b34801561033957600080fd5b50610342610ec1565b6040518082815260200191505060405180910390f35b34801561036457600080fd5b506103c3600480360381019080803573ffffffffffffffffffffffffffffffffffffffff169060200190929190803573ffffffffffffffffffffffffffffffffffffffff16906020019092919080359060200190929190505050610fa9565b005b3480156103d157600080fd5b506103da61118e565b604051808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390f35b34801561042857600080fd5b5061045d600480360381019080803573ffffffffffffffffffffffffffffffffffffffff1690602001909291905050506111b4565b6040518082815260200191505060405180910390f35b34801561047f57600080fd5b506104886111cc565b6040518082815260200191505060405180910390f35b3480156104aa57600080fd5b506104b36111d2565b6040518082815260200191505060405180910390f35b3480156104d557600080fd5b506104de6111d8565b6040518082815260200191505060405180910390f35b34801561050057600080fd5b506105096111de565b005b34801561051757600080fd5b5061054c600480360381019080803573ffffffffffffffffffffffffffffffffffffffff16906020019092919050505061129c565b604051808215151515815260200191505060405180910390f35b34801561057257600080fd5b506105c7600480360381019080803573ffffffffffffffffffffffffffffffffffffffff169060200190929190803573ffffffffffffffffffffffffffffffffffffffff1690602001909291905050506112f2565b6040518082815260200191505060405180910390f35b3480156105e957600080fd5b506105f2611317565b604051808215151515815260200191505060405180910390f35b34801561061857600080fd5b5061064d600480360381019080803573ffffffffffffffffffffffffffffffffffffffff16906020019092919050505061132a565b6040518082815260200191505060405180910390f35b34801561066f57600080fd5b50610678611451565b005b34801561068657600080fd5b5061068f611511565b604051808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390f35b3480156106dd57600080fd5b506106e661153a565b604051808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390f35b34801561073457600080fd5b5061073d61155f565b6040518080602001828103825283818151815260200191508051906020019080838360005b8381101561077d578082015181840152602081019050610762565b50505050905090810190601f1680156107aa5780820380516001836020036101000a031916815260200191505b509250505060405180910390f35b3480156107c457600080fd5b50610803600480360381019080803573ffffffffffffffffffffffffffffffffffffffff169060200190929190803590602001909291905050506115fd565b005b34801561081157600080fd5b5061083a60048036038101908080359060200190929190803590602001909291905050506117ac565b005b34801561084857600080fd5b5061086760048036038101908080359060200190929190505050611891565b005b34801561087557600080fd5b5061089460048036038101908080359060200190929190505050611a88565b005b3480156108a257600080fd5b506108f7600480360381019080803573ffffffffffffffffffffffffffffffffffffffff169060200190929190803573ffffffffffffffffffffffffffffffffffffffff169060200190929190505050611c1b565b6040518082815260200191505060405180910390f35b34801561091957600080fd5b50610922611d78565b6040518082815260200191505060405180910390f35b34801561094457600080fd5b50610979600480360381019080803573ffffffffffffffffffffffffffffffffffffffff169060200190929190505050611d7e565b604051808215151515815260200191505060405180910390f35b34801561099f57600080fd5b506109d4600480360381019080803573ffffffffffffffffffffffffffffffffffffffff169060200190929190505050611d9e565b005b3480156109e257600080fd5b506109eb611eb7565b6040518082815260200191505060405180910390f35b348015610a0d57600080fd5b50610a42600480360381019080803573ffffffffffffffffffffffffffffffffffffffff169060200190929190505050611edb565b005b348015610a5057600080fd5b50610a85600480360381019080803573ffffffffffffffffffffffffffffffffffffffff169060200190929190505050611fb0565b005b60078054600181600116156101000203166002900480601f016020809104026020016040519081016040528092919081815260200182805460018160011615610100020316600290048015610b1d5780601f10610af257610100808354040283529160200191610b1d565b820191906000526020600020905b815481529060010190602001808311610b0057829003601f168201915b505050505081565b6000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff16141515610b8057600080fd5b6001600a60146101000a81548160ff02191690831515021790555080600a60006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff1602179055507fcc358699805e9a8b7f77b522628c7cb9abd07d9efb86b6fb616af1609036a99e81604051808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390a150565b604060048101600036905010151515610c5a57600080fd5b600a60149054906101000a900460ff1615610d8557600a60009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1663aee92d333385856040518463ffffffff167c0100000000000000000000000000000000000000000000000000000000028152600401808473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020018373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020018281526020019350505050600060405180830381600087803b158015610d6857600080fd5b505af1158015610d7c573d6000803e3d6000fd5b50505050610d90565b610d8f8383612134565b5b505050565b600a60149054906101000a900460ff1681565b6000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff16141515610e0357600080fd5b6001600660008373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060006101000a81548160ff0219169083151502179055507f42e160154868087d6bfdc0ca23d96a1c1cfa32f1b72ba9ba27b69b98a0d819dc81604051808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390a150565b6000600a60149054906101000a900460ff1615610fa057600a60009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff166318160ddd6040518163ffffffff167c0100000000000000000000000000000000000000000000000000000000028152600401602060405180830381600087803b158015610f5e57600080fd5b505af1158015610f72573d6000803e3d6000fd5b505050506040513d6020811015610f8857600080fd5b81019080805190602001909291905050509050610fa6565b60015490505b90565b600060149054906101000a900460ff16151515610fc557600080fd5b600660008473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060009054906101000a900460ff1615151561101e57600080fd5b600a60149054906101000a900460ff161561117d57600a60009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16638b477adb338585856040518563ffffffff167c0100000000000000000000000000000000000000000000000000000000028152600401808573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020018473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020018373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001828152602001945050505050600060405180830381600087803b15801561116057600080fd5b505af1158015611174573d6000803e3d6000fd5b50505050611189565b6111888383836122d1565b5b505050565b600a60009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1681565b60026020528060005260406000206000915090505481565b60095481565b60045481565b60015481565b6000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff1614151561123957600080fd5b600060149054906101000a900460ff16151561125457600080fd5b60008060146101000a81548160ff0219169083151502179055507f7805862f689e2f13df9f062ff482ad3ad112aca9e0847911ed832e158c525b3360405160405180910390a1565b6000600660008373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060009054906101000a900460ff169050919050565b6005602052816000526040600020602052806000526040600020600091509150505481565b600060149054906101000a900460ff1681565b6000600a60149054906101000a900460ff161561144057600a60009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff166370a08231836040518263ffffffff167c0100000000000000000000000000000000000000000000000000000000028152600401808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001915050602060405180830381600087803b1580156113fe57600080fd5b505af1158015611412573d6000803e3d6000fd5b505050506040513d602081101561142857600080fd5b8101908080519060200190929190505050905061144c565b61144982612778565b90505b919050565b6000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff161415156114ac57600080fd5b600060149054906101000a900460ff161515156114c857600080fd5b6001600060146101000a81548160ff0219169083151502179055507f6985a02210a168e66602d3235cb6db0e70f92b3ba4d376a33c0f3d9434bff62560405160405180910390a1565b60008060009054906101000a900473ffffffffffffffffffffffffffffffffffffffff16905090565b6000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff1681565b60088054600181600116156101000203166002900480601f0160208091040260200160405190810160405280929190818152602001828054600181600116156101000203166002900480156115f55780601f106115ca576101008083540402835291602001916115f5565b820191906000526020600020905b8154815290600101906020018083116115d857829003601f168201915b505050505081565b600060149054906101000a900460ff1615151561161957600080fd5b600660003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060009054906101000a900460ff1615151561167257600080fd5b600a60149054906101000a900460ff161561179d57600a60009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16636e18980a3384846040518463ffffffff167c0100000000000000000000000000000000000000000000000000000000028152600401808473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020018373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020018281526020019350505050600060405180830381600087803b15801561178057600080fd5b505af1158015611794573d6000803e3d6000fd5b505050506117a8565b6117a782826127c1565b5b5050565b6000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff1614151561180757600080fd5b60148210151561181657600080fd5b60328110151561182557600080fd5b81600381905550611844600954600a0a82612b2990919063ffffffff16565b6004819055507fb044a1e409eac5c48e5af22d4af52670dd1a99059537a78b31b48c6500a6354e600354600454604051808381526020018281526020019250505060405180910390a15050565b6000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff161415156118ec57600080fd5b600154816001540111151561190057600080fd5b600260008060009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000205481600260008060009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002054011115156119d057600080fd5b80600260008060009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060008282540192505081905550806001600082825401925050819055507fcb8241adb0c3fdb35b70c24ce35c5eb0c17af7431c99f827d44a445ca624176a816040518082815260200191505060405180910390a150565b6000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff16141515611ae357600080fd5b8060015410151515611af457600080fd5b80600260008060009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000205410151515611b6357600080fd5b8060016000828254039250508190555080600260008060009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020600082825403925050819055507f702d5967f45f6513a38ffc42d6ba9bf230bd40e8f53b16363c7eb4fd2deb9a44816040518082815260200191505060405180910390a150565b6000600a60149054906101000a900460ff1615611d6557600a60009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1663dd62ed3e84846040518363ffffffff167c0100000000000000000000000000000000000000000000000000000000028152600401808373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020018273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200192505050602060405180830381600087803b158015611d2357600080fd5b505af1158015611d37573d6000803e3d6000fd5b505050506040513d6020811015611d4d57600080fd5b81019080805190602001909291905050509050611d72565b611d6f8383612b64565b90505b92915050565b60035481565b60066020528060005260406000206000915054906101000a900460ff1681565b6000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff16141515611df957600080fd5b6000600660008373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060006101000a81548160ff0219169083151502179055507fd7e9ec6e6ecd65492dce6bf513cd6867560d49544421d0783ddf06e76c24470c81604051808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390a150565b7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff81565b6000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff16141515611f3657600080fd5b600073ffffffffffffffffffffffffffffffffffffffff168173ffffffffffffffffffffffffffffffffffffffff16141515611fad57806000806101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff1602179055505b50565b60008060009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff1614151561200d57600080fd5b600660008373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060009054906101000a900460ff16151561206557600080fd5b61206e8261132a565b90506000600260008473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002081905550806001600082825403925050819055507f61e6e66b0d6339b2980aecc6ccc0039736791f0ccde9ed512e789a7fbdd698c68282604051808373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020018281526020019250505060405180910390a15050565b60406004810160003690501015151561214c57600080fd5b600082141580156121da57506000600560003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060008573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000205414155b1515156121e657600080fd5b81600560003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060008573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020819055508273ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff167f8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925846040518082815260200191505060405180910390a3505050565b60008060006060600481016000369050101515156122ee57600080fd5b600560008873ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002054935061239661271061238860035488612b2990919063ffffffff16565b612beb90919063ffffffff16565b92506004548311156123a85760045492505b7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff841015612464576123e38585612c0690919063ffffffff16565b600560008973ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020819055505b6124778386612c0690919063ffffffff16565b91506124cb85600260008a73ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002054612c0690919063ffffffff16565b600260008973ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000208190555061256082600260008973ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002054612c1f90919063ffffffff16565b600260008873ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002081905550600083111561270a5761261f83600260008060009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002054612c1f90919063ffffffff16565b600260008060009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020819055506000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168773ffffffffffffffffffffffffffffffffffffffff167fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef856040518082815260200191505060405180910390a35b8573ffffffffffffffffffffffffffffffffffffffff168773ffffffffffffffffffffffffffffffffffffffff167fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef846040518082815260200191505060405180910390a350505050505050565b6000600260008373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020549050919050565b6000806040600481016000369050101515156127dc57600080fd5b6128056127106127f760035487612b2990919063ffffffff16565b612beb90919063ffffffff16565b92506004548311156128175760045492505b61282a8385612c0690919063ffffffff16565b915061287e84600260003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002054612c0690919063ffffffff16565b600260003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000208190555061291382600260008873ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002054612c1f90919063ffffffff16565b600260008773ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020819055506000831115612abd576129d283600260008060009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002054612c1f90919063ffffffff16565b600260008060009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020819055506000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff167fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef856040518082815260200191505060405180910390a35b8473ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff167fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef846040518082815260200191505060405180910390a35050505050565b6000806000841415612b3e5760009150612b5d565b8284029050828482811515612b4f57fe5b04141515612b5957fe5b8091505b5092915050565b6000600560008473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060008373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002054905092915050565b6000808284811515612bf957fe5b0490508091505092915050565b6000828211151515612c1457fe5b818303905092915050565b6000808284019050838110151515612c3357fe5b80915050929150505600a165627a7a72305820e4fc3a34d58644d5864db1d059c2745140e4b65f6dbc3d50bf901bbeeca21f130029000000000000000000000000000000000000000000000000002386f26fc10000000000000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000000000c00000000000000000000000000000000000000000000000000000000000000006000000000000000000000000000000000000000000000000000000000000000a757364745f746f6b656e0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000047573647400000000000000000000000000000000000000000000000000000000"

type GenesisProposer struct {
	MinerId     string `yaml:"minerId"`
	MinerPubKey string `yaml:"minerPubkey"`
	VRFPubkey   string `yaml:"vrfPubkey"`
}

func (chain *blockChain) insertGenesisBlock() {
	state, err := service.AccountDBManagerInstance.GetAccountDBByHash(common.Hash{})
	if nil == err {
		genesisBlock := genGenesisBlock(state, service.AccountDBManagerInstance.GetTrieDB(), consensusHelper.GenerateGenesisInfo())
		logger.Debugf("GenesisBlock Hash:%s,StateTree:%s", genesisBlock.Header.Hash.String(), genesisBlock.Header.StateTree.Hex())
		blockByte, _ := types.MarshalBlock(genesisBlock)
		chain.saveBlockByHash(genesisBlock.Header.Hash, blockByte)

		headerByte, err := types.MarshalBlockHeader(genesisBlock.Header)
		if err != nil {
			logger.Errorf("Marshal block header error:%s", err.Error())
		}
		chain.saveBlockByHeight(genesisBlock.Header.Height, headerByte)

		chain.updateLastBlock(state, genesisBlock, headerByte)
		chain.updateVerifyHash(genesisBlock)
	} else {
		panic("Init block chain error:" + err.Error())
	}
}

func genGenesisBlock(stateDB *account.AccountDB, triedb *trie.NodeDatabase, genesisInfo []*types.GenesisInfo) *types.Block {
	block := new(types.Block)
	pv := big.NewInt(0)
	block.Header = &types.BlockHeader{
		Height:       0,
		ExtraData:    common.Sha256([]byte("Rocket Protocol")),
		CurTime:      time.Date(2020, 12, 21, 10, 0, 0, 0, time.UTC),
		ProveValue:   pv,
		TotalQN:      0,
		Transactions: make([]common.Hashes, 0), //important!!
		EvictedTxs:   make([]common.Hash, 0),   //important!!
		Nonce:        ChainDataVersion,
	}

	block.Header.RequestIds = make(map[string]uint64)
	block.Header.Signature = common.Sha256([]byte("tuntunhz"))
	block.Header.Random = common.Sha256([]byte("RocketProtocolVRF"))

	genesisProposers := getGenesisProposer()
	addMiners(genesisProposers, stateDB)

	verifyMiners := make([]*types.Miner, 0)
	for _, genesis := range genesisInfo {
		for i, member := range genesis.Group.Members {
			miner := &types.Miner{Type: common.MinerTypeValidator, Id: member, PublicKey: genesis.Pks[i], VrfPublicKey: genesis.VrfPKs[i], Stake: common.ValidatorStake * uint64(i+2)}
			verifyMiners = append(verifyMiners, miner)
		}
	}
	addMiners(verifyMiners, stateDB)

	//addTestMiners(stateDB)

	stateDB.SetNonce(common.ProposerDBAddress, 1)
	stateDB.SetNonce(common.ValidatorDBAddress, 1)

	valueTen, _ := utility.StrToBigInt("10")
	valueBillion, _ := utility.StrToBigInt("1000000000")

	//创建创始合约
	usdtContractAddress := createGenesisContract(block.Header, stateDB)
	stateDB.AddERC20Binding("SYSTEM-ETH.USDT", usdtContractAddress, 2, 6)

	// 测试用
	service.FTManagerInstance.PublishFTSet(service.FTManagerInstance.GenerateFTSet("tuntun", "pig", "hz", "0", "hz", "10086", 0), stateDB)
	service.NFTManagerInstance.PublishNFTSet(service.NFTManagerInstance.GenerateNFTSet("tuntunhz", "tuntun", "t", "hz", "hz", types.NFTConditions{}, 0, "10000"), stateDB)
	stateDB.SetBNT(common.HexToAddress("0x0b7467fe7225e8adcb6b5779d68c20fceaa58d54"), "ETH.ETH", valueTen)
	/**
		测试账户
		id:0x6420e467c77514e09471a7d84e0552c13b5e97192f523c05d3970d7ee23bf443
	    adderss:0x38780174572fb5b4735df1b7c69aee77ff6e9f49
	    sk:0xe7260a418579c2e6ca36db4fe0bf70f84d687bdf7ec6c0c181b43ee096a84aea
	*/
	stateDB.SetBNT(common.HexToAddress("0x38780174572fb5b4735df1b7c69aee77ff6e9f49"), "ETH.ETH", valueTen)
	stateDB.SetBalance(common.HexToAddress("0x38780174572fb5b4735df1b7c69aee77ff6e9f49"), valueBillion)

	//客户端测试使用账号
	stateDB.SetBNT(common.HexToAddress("0xe7260a418579c2e6ca36db4fe0bf70f84d687bdf7ec6c0c181b43ee096a84aea"), "ETH.ETH", valueTen)
	stateDB.SetFT(common.HexToAddress("0xe7260a418579c2e6ca36db4fe0bf70f84d687bdf7ec6c0c181b43ee096a84aea"), "SYSTEM-ETH.USDT", valueTen)
	stateDB.SetBalance(common.HexToAddress("0xe7260a418579c2e6ca36db4fe0bf70f84d687bdf7ec6c0c181b43ee096a84aea"), valueBillion)

	stateDB.SetBalance(common.HexToAddress("0xa72263e15d3a48de8cbde1f75c889c85a15567b990aab402691938f4511581ab"), valueBillion)
	/**
	自动化测试框架专用账户
	id:0x7dba6865f337148e5887d6bea97e6a98701a2fa774bd00474ea68bcc645142f2
	address:0x2c616a97d3d10e008f901b392986b1a65e0abbb7
	sk:0x083f3fb13ffa99a18283a7fd5e2f831a52f39afdd90f5310a3d8fd4ffbd00d49
	*/
	stateDB.SetBNT(common.HexToAddress("0x2c616a97d3d10e008f901b392986b1a65e0abbb7"), "ETH.ETH", valueTen)
	stateDB.SetFT(common.HexToAddress("0x2c616a97d3d10e008f901b392986b1a65e0abbb7"), "SYSTEM-ETH.USDT", valueTen)
	stateDB.SetBalance(common.HexToAddress("0x2c616a97d3d10e008f901b392986b1a65e0abbb7"), valueBillion)

	root, _ := stateDB.Commit(true)
	triedb.Commit(root, false)
	block.Header.StateTree = common.BytesToHash(root.Bytes())
	block.Header.Hash = block.Header.GenHash()

	return block
}

func getGenesisProposer() []*types.Miner {
	genesisProposers := make([]GenesisProposer, 1)
	genesisProposer := GenesisProposer{}
	genesisProposer.MinerId = "0x7f88b4f2d36a83640ce5d782a0a20cc2b233de3df2d8a358bf0e7b29e9586a12"
	genesisProposer.MinerPubKey = "0x16d0b0a106e2de32b42ea4096c9e80c883c6ffa9e3f19f09cb45dfff2b02d09a3bcf95f2d0c33b7caf5db42d55d3459395c1b8d6a5d315a113edc39c4ce3a3d5269ab4a9514a998fdcc693d90a42505185270a184a07ddfb553b181be13e968480ef0df4c06cf657957b07118776a38fea3bcf758ea4491a4213719e2f6537b5"
	genesisProposer.VRFPubkey = "0x009f3b76f3e49dcdd6d2ee8421f077fd4c68c176b18e1e602a3c1f09f9272250"
	genesisProposers[0] = genesisProposer

	miners := make([]*types.Miner, 0)
	for _, gp := range genesisProposers {
		var minerId groupsig.ID
		minerId.SetHexString(gp.MinerId)

		var minerPubkey groupsig.Pubkey
		minerPubkey.SetHexString(gp.MinerPubKey)

		vrfPubkey := vrf.Hex2VRFPublicKey(gp.VRFPubkey)
		miner := types.Miner{
			Id:           minerId.Serialize(),
			PublicKey:    minerPubkey.Serialize(),
			VrfPublicKey: vrfPubkey,
			ApplyHeight:  0,
			Stake:        common.ProposerStake,
			Type:         common.MinerTypeProposer,
			Status:       common.MinerStatusNormal,
		}
		miners = append(miners, &miner)
	}
	return miners
}

func addMiners(miners []*types.Miner, accountdb *account.AccountDB) {
	for _, miner := range miners {
		service.MinerManagerImpl.InsertMiner(miner, accountdb)
	}
}

func createGenesisContract(header *types.BlockHeader, statedb *account.AccountDB) common.Address {
	source := "0x38780174572fb5b4735df1b7c69aee77ff6e9f49"
	vmCtx := vm.Context{}
	vmCtx.CanTransfer = vm.CanTransfer
	vmCtx.Transfer = vm.Transfer
	vmCtx.GetHash = func(uint64) common.Hash { return emptyHash }

	vmCtx.Origin = common.HexToAddress(source)
	vmCtx.Coinbase = common.BytesToAddress(header.Castor)
	vmCtx.BlockNumber = new(big.Int).SetUint64(header.Height)
	vmCtx.Time = new(big.Int).SetUint64(uint64(header.CurTime.Unix()))
	vmCtx.GasPrice = big.NewInt(1)
	vmCtx.GasLimit = 30000000

	vmInstance := vm.NewEVM(vmCtx, statedb)
	caller := vm.AccountRef(vmCtx.Origin)

	_, usdtContractAddress, _, _, err := vmInstance.Create(caller, common.FromHex(usdtContractData), vmCtx.GasLimit, big.NewInt(0))
	if err != nil {
		panic("Genesis contract create error:" + err.Error())
	}
	logger.Debugf("After execute usdt contract create!Contract address:%s", usdtContractAddress.GetHexString())
	return usdtContractAddress
}
