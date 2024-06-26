// Copyright 2020 The RangersProtocol Authors
// This file is part of the RocketProtocol library.
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

package cli

import (
	"com.tuntun.rangers/node/src/common"
	"com.tuntun.rangers/node/src/consensus"
	"com.tuntun.rangers/node/src/consensus/logical/group_create"
	"com.tuntun.rangers/node/src/consensus/model"
	"com.tuntun.rangers/node/src/core"
	"com.tuntun.rangers/node/src/middleware"
	"com.tuntun.rangers/node/src/middleware/types"
	"com.tuntun.rangers/node/src/service"
	"com.tuntun.rangers/node/src/storage/account"
	"com.tuntun.rangers/node/src/vm"
	"fmt"
	"math/big"
	"os"
	"testing"
	"time"
)

const padding = "0000000000000000000000000000000000000000000000000000000000000060"

func TestMinerEcomony(t *testing.T) {
	defer func() {
		os.RemoveAll("storage0")
		os.RemoveAll("logs")
		os.RemoveAll("1.ini")
		os.RemoveAll("genesis.json")
	}()

	os.WriteFile("genesis.json", []byte(`{"chainId":"9000","name":"testsub","cast":2000,"groupLife":200000,"p":2,"v":3,"genesisTime":1672815214204,"timecycle":100,"tokenName":"mycoin","totalsupply":21000000,"symbol":"mc","decimal":0,"ptoken":30,"vtoken":30,"group":"{\"GroupInfo\":{\"GroupID\":\"0x36678f8a0682b8e6ce79e868f9ecd5e20c66df99c7fc9f6dd1b646dc43c4f58c\",\"GroupPK\":\"0x2695a9faa002f4e079cef76990e6dbb8ea591831cac8abe18d666ccb75e97f6b8d44a6fb29acdb11f1ee040d79a675bff30e9db16b50c5d06aa27575d5cc59041d20b08e8af1f7c133782255d25b2d5bcf9d8135c4ddf08d90238168b006377c603d85c91c49d6d54d7ed645b5fd9abd187936955a2b19c6fb96d7c466acbc1a\",\"GroupInitInfo\":{\"GroupHeader\":{\"Hash\":\"0x0cad787b74f184febdec0ab2a4c7269d0112e66e00cbda2f000f27323b21a6f0\",\"Parent\":null,\"PreGroup\":null,\"Authority\":777,\"Name\":\"Genesis Group\",\"BeginTime\":\"2023-01-04T14:53:48.80013+08:00\",\"MemberRoot\":\"0xb0be0dc911294b2ed78e0a577691c711c8958bc8973b96c3b2c4cfe31a7a666f\",\"CreateHeight\":0,\"ReadyHeight\":1,\"WorkHeight\":0,\"DismissHeight\":18446744073709551615,\"Extends\":\"\"},\"ParentGroupSign\":{},\"GroupMembers\":[\"0x6a58a9438e2e22d585d6965665b7482a91f9d1f2bcba567e97121a351369c1f0\",\"0xf0666387d32ae728ad2d046f32978b6e521152a2dab2c87e91d2f11a7cb17c29\",\"0x96feef21cdacb86d9d56099763de643ebe30e65f17f13e6d7c8931a794079d14\"]},\"MemberIndexMap\":{\"0x6a58a9438e2e22d585d6965665b7482a91f9d1f2bcba567e97121a351369c1f0\":0,\"0x96feef21cdacb86d9d56099763de643ebe30e65f17f13e6d7c8931a794079d14\":2,\"0xf0666387d32ae728ad2d046f32978b6e521152a2dab2c87e91d2f11a7cb17c29\":1},\"ParentGroupID\":\"0x0000000000000000000000000000000000000000000000000000000000000000\",\"PrevGroupID\":\"0x0000000000000000000000000000000000000000000000000000000000000000\"},\"VrfPubkey\":[\"GzAWP6M9xpuek2FYl/urhAui6Hg/S6kQRyCCMSSE/Lw=\",\"dClZW4ovDiEwOTA9z9AP5zhCSr+aZdvhmkhrnB8ZiGk=\",\"+yB3pMwOpfj3PG/2FT3xljwXy6pukC1BNQhMV88TNqo=\"],\"Pubkeys\":[\"0x1d1105553d0858f1ed275a6c7e1b21f2a8a3e423c4906e58a209e71a4b2853d58b53a4922e6a437204fba3a4b63c6b2627a0b4e94f556c40863a91dc2bcce2dd51c4e76b1aa427f0f646db53b9caa869aa3c341495bdf36f271cd48e0f4676033bfa76bf200ef978ce53f326e1033d3a090d891bcb38ba849c2e61e1e6a4a8da\",\"0x54c25070f01afe847340bf25ed95b0d7c2c42c35ccd858a22ab0d3399b1bb47f6524f7219652206a895206d49e2dd535aff7a2aeec0f2da78b9a1b9c8fa0bcce1b9a3d373a32e2d660cd11e6adddf06eef98f740c24d049b91e3ff5a52ceedd055e681bde5c54b175a1b49cca2c3c04a43e197412368912844e5c49fe2bb84fe\",\"0x378e973c44b7c4862926a9c4277b3d3c09dd77606abd7781bd1efd1dd1cd2c8f10d91e23d9a59d63c2e4428a50adcdda198d063203554c9558cd1a2aae2df8de7d83b1f8cb8e53ffadd29b8ad1b5d699cb0802a1ccf7bdd0191a8fcc49b3c2517bcfc6c52d2699c03756b59897bf16b1e78f319c12e95fea2d016ad422203902\"]}","joined":"{\"GroupHash\":\"0x0cad787b74f184febdec0ab2a4c7269d0112e66e00cbda2f000f27323b21a6f0\",\"GroupID\":\"0x36678f8a0682b8e6ce79e868f9ecd5e20c66df99c7fc9f6dd1b646dc43c4f58c\",\"GroupPK\":\"0x2695a9faa002f4e079cef76990e6dbb8ea591831cac8abe18d666ccb75e97f6b8d44a6fb29acdb11f1ee040d79a675bff30e9db16b50c5d06aa27575d5cc59041d20b08e8af1f7c133782255d25b2d5bcf9d8135c4ddf08d90238168b006377c603d85c91c49d6d54d7ed645b5fd9abd187936955a2b19c6fb96d7c466acbc1a\",\"SignSecKey\":{},\"MemberSignPubkeyMap\":{\"0x6a58a9438e2e22d585d6965665b7482a91f9d1f2bcba567e97121a351369c1f0\":\"0x3992e17121a373346dc191043c3726681a83786afd6cf3197d1cef472379bbaa23c2b9cddaf6e5d4a964bccaf483c617037313a1dbc1c5a1d4a877ac15a097ba4c4ae59eff0cdeb7b1b8cf9f893eb4ad9fe68016493e44497498501d06bff1a9840fb004558834e4b220880720963afed76d3e8d3447c2cc5a6bf4c379159dee\",\"0x96feef21cdacb86d9d56099763de643ebe30e65f17f13e6d7c8931a794079d14\":\"0x15700ca8352fcdfd3ccb0fad1ddfeef04d127b5a83468796cd3cb95100d22112851db3bb59ae435ede3f78fa9fa9fc57bc7b69a2dcba56ba071521a3f0c1a6c077bfa37216bd87eb5331fafacca230ed5d8f84b66ebbcdc31ec540afb759d0574c9992de239c97ca5a6eec8e63fdc994d40dad17342efd2f320aaec009942715\",\"0xf0666387d32ae728ad2d046f32978b6e521152a2dab2c87e91d2f11a7cb17c29\":\"0x16c9d999a4ee7d9d9621d893beb327b994818b962f16220fc6ee93b03b2a6a7760607589f294605d79184f2d6e78caa8a676f241097d7142ee1f8626084c54a7164480cd891eb950888b623f12e66631d3e6669d4048837cf80581b36195e7797f5f44604fa09171cfc4003bb0c6f774b4d905278c5cca29f3d7038edeed336b\"}}","proposers":["{\"account\":\"0x908c6d839c5a00eb4a035f1357a798ed6fc07ef6\",\"id\":\"0x6ac496268d1102bb570853a2908c6d839c5a00eb4a035f1357a798ed6fc07ef6\",\"publicKey\":\"0x8f31c75fd3a12f546fe7a1b854fc744c6fa74f91412b2d1d52b439b52e8f980d499eeecd8143a8e4d40c31c650ab99d5136333112773d7cbbcb9c7c4db2efd7c30668698127bb4902dfe77ca271a2150bddbeac5b03b1d9c589b6cc891d045bd728b25801bdbb4d5db9e425240e5ba58618dc7017ad7d26ee5bdddfe2b755b72\",\"vrfPublicKey\":\"0xaacb68b726ab57b545110dda8b99e51872f31b2ebd845651174ca3f125ca1cff\"}","{\"account\":\"0x9c0e374bf6fba1db2d767c5dc27342a6d200d48a\",\"id\":\"0x5bc45d7d229f7afd9aa190e99c0e374bf6fba1db2d767c5dc27342a6d200d48a\",\"publicKey\":\"0x09162936a69808f12d23bf4a46cbdea1e75c1bc17829e82bbbffcb4f9637c78d6c9021987c5f162b653d9584deba04dcde91a73b5a631cc742338920de3ca8e76a822db1a2af61dee392296db2d763cc124edbb9f09fd8d1268cb6dfd282d331674077741ce3811e1d65efda1e7d9eccc9d8ae4e3b081ad3fd47e44bf66d3fb4\",\"vrfPublicKey\":\"0x0966b53bd92b4f93345b5a260b74d5fa0563c8e25fc3f1f5605aba37e4678e99\"}"]}`), 0766)
	initTestingEnv()

	block := new(types.Block)
	pv := big.NewInt(0)
	block.Header = &types.BlockHeader{
		Height:       1,
		ProveValue:   pv,
		TotalQN:      0,
		Transactions: make([]common.Hashes, 0), //important!!
		EvictedTxs:   make([]common.Hash, 0),   //important!!
		Nonce:        2,
		CurTime:      time.Now(),
	}

	group := core.GetGroupChain().GetGroupByHeight(0)
	if nil == group {
		panic("no genesis group")
	}
	block.Header.GroupId = group.Id
	block.Header.Castor = common.FromHex("0x6ac496268d1102bb570853a2908c6d839c5a00eb4a035f1357a798ed6fc07ef6")

	header := block.Header
	vmCtx := vm.Context{}
	vmCtx.CanTransfer = vm.CanTransfer
	vmCtx.Transfer = transfer
	vmCtx.GetHash = func(uint64) common.Hash { return common.Hash{} }
	vmCtx.Origin = common.HexToAddress("0x1111111111111111111111111111111111111111")
	vmCtx.Coinbase = common.BytesToAddress(header.Castor)
	vmCtx.BlockNumber = new(big.Int).SetUint64(header.Height)
	vmCtx.Time = new(big.Int).SetUint64(uint64(header.CurTime.Unix()))
	vmCtx.GasPrice = big.NewInt(1)
	vmCtx.GasLimit = 30000000

	accountdb := middleware.AccountDBManagerInstance.GetLatestStateDB()
	fmt.Print("0x908c6d839c5a00eb4a035f1357a798ed6fc07ef6: ")
	fmt.Println(accountdb.GetBalance(common.HexToAddress("0x908c6d839c5a00eb4a035f1357a798ed6fc07ef6")))
	fmt.Print("0x9c0e374bf6fba1db2d767c5dc27342a6d200d48a: ")
	fmt.Println(accountdb.GetBalance(common.HexToAddress("0x9c0e374bf6fba1db2d767c5dc27342a6d200d48a")))
	fmt.Print("0x0e05d86e7943d7f041fabde02f25d53a2aa4cc29: ")
	fmt.Println(accountdb.GetBalance(common.HexToAddress("0x0e05d86e7943d7f041fabde02f25d53a2aa4cc29")))
	fmt.Print("0xe9b59d7af13bf6d3f838da7f73c2e369802ea211: ")
	fmt.Println(accountdb.GetBalance(common.HexToAddress("0xe9b59d7af13bf6d3f838da7f73c2e369802ea211")))
	fmt.Print("0x1aab2207e31dff81240fc4976c301ab0a0e0da26: ")
	fmt.Println(accountdb.GetBalance(common.HexToAddress("0x1aab2207e31dff81240fc4976c301ab0a0e0da26")))

	start := time.Now().UnixMilli()
	vmInstance := vm.NewEVMWithNFT(vmCtx, accountdb, accountdb)
	caller := vm.AccountRef(vmCtx.Origin)
	code := generateCode(header, accountdb)
	codeBytes := common.FromHex(code)
	_, _, _, err := vmInstance.Call(caller, common.EconomyContract, codeBytes, vmCtx.GasLimit, big.NewInt(0))
	if nil != err {
		t.Fatal(err)
	}
	fmt.Printf("time using: %d\n", time.Now().UnixMilli()-start)

	fmt.Print("0x908c6d839c5a00eb4a035f1357a798ed6fc07ef6: ")
	fmt.Println(accountdb.GetBalance(common.HexToAddress("0x908c6d839c5a00eb4a035f1357a798ed6fc07ef6")))
	fmt.Print("0x9c0e374bf6fba1db2d767c5dc27342a6d200d48a: ")
	fmt.Println(accountdb.GetBalance(common.HexToAddress("0x9c0e374bf6fba1db2d767c5dc27342a6d200d48a")))
	fmt.Print("0x0e05d86e7943d7f041fabde02f25d53a2aa4cc29: ")
	fmt.Println(accountdb.GetBalance(common.HexToAddress("0x0e05d86e7943d7f041fabde02f25d53a2aa4cc29")))
	fmt.Print("0xe9b59d7af13bf6d3f838da7f73c2e369802ea211: ")
	fmt.Println(accountdb.GetBalance(common.HexToAddress("0xe9b59d7af13bf6d3f838da7f73c2e369802ea211")))
	fmt.Print("0x1aab2207e31dff81240fc4976c301ab0a0e0da26: ")
	fmt.Println(accountdb.GetBalance(common.HexToAddress("0x1aab2207e31dff81240fc4976c301ab0a0e0da26")))

	codeBytes = common.FromHex("0x70a082310000000000000000000000001aab2207e31dff81240fc4976c301ab0a0e0da26")
	ret, _, _, err := vmInstance.Call(caller, common.HexToAddress("0x1ca05267f79ad1e496956922f257c2ff6c8e1892"), codeBytes, vmCtx.GasLimit, big.NewInt(0))
	if nil != err {
		t.Fatal(err)
	}
	bigInt := big.NewInt(0).SetBytes(ret)
	fmt.Println(bigInt.String())

}

func generateCode(header *types.BlockHeader, accountdb *account.AccountDB) string {
	proposals, validators := service.MinerManagerImpl.GetAllMinerIdAndAccount(header.Height, accountdb)

	code := "0x7822b9ac" + common.GenerateCallDataAddress(proposals[common.ToHex(header.Castor)]) + padding + common.GenerateCallDataUint(uint64(4+len(proposals))*32)
	code += common.GenerateCallDataUint(uint64(len(proposals)))
	for _, addr := range proposals {
		code += common.GenerateCallDataAddress(addr)
	}

	// get validator group
	groupId := header.GroupId
	group := core.GetGroupChain().GetGroupById(groupId)
	if group == nil {
		return ""
	}

	code += common.GenerateCallDataUint(uint64(len(group.Members)))
	for _, member := range group.Members {
		code += common.GenerateCallDataAddress(validators[common.ToHex(member)])
	}

	return code
}

// Transfer subtracts amount from sender and adds amount to recipient using the given Db
func transfer(db vm.StateDB, sender, recipient common.Address, amount *big.Int) {
	if nil == amount || 0 == amount.Sign() {
		return
	}

	fmt.Printf("sender: %s, recipient: %s, amount: %s\n", sender.String(), recipient.String(), amount.String())
	db.SubBalance(sender, amount)
	db.AddBalance(recipient, amount)
}

func initTestingEnv() {
	common.Init(0, "1.ini", "dev")

	middleware.InitMiddleware()

	privateKey := common.GenerateKey("")
	account := getAccountByPrivateKey(privateKey.GetHexString())
	fmt.Println("Your Miner Address:", account.Address)

	sk := common.HexStringToSecKey(account.Sk)
	minerInfo := model.NewSelfMinerInfo(*sk)
	common.GlobalConf.SetString(Section, "miner", minerInfo.ID.GetHexString())

	service.InitService()
	vm.InitVM()

	err := core.InitCore(consensus.NewConsensusHelper(minerInfo.ID), *sk, minerInfo.ID.GetHexString())
	if err != nil {
		panic("Init miner core init error:" + err.Error())
	}

	ok := consensus.InitConsensus(minerInfo, common.GlobalConf)
	if !ok {
		panic("Init miner consensus init error!")

	}

	//consensus.Proc.BeginGenesisGroupMember()
	group_create.GroupCreateProcessor.BeginGenesisGroupMember()
}

func TestNewGX(t *testing.T) {
	gx := NewGX()

	common.Init(0, "1.ini", "dev")
	gx.initMiner("dev", "ws://192.168.2.14:1017", "ws://192.168.2.14:1213", "ws://192.168.2.14:8888")

	time.Sleep(10 * time.Hour)
}

func TestMiner(t *testing.T) {
	privateKey := common.GenerateKey("")
	mi := model.NewSelfMinerInfo(privateKey)

	fmt.Println(privateKey.GetHexString())

	//0x7f88b4f2d36a83640ce5d782a0a20cc2b233de3df2d8a358bf0e7b29e9586a12
	//0xc97985713ef6ef7f59a39a50e1e8a089c2619b7a280d024f45663ece8fe4b30e
	fmt.Println(mi.ID.GetHexString())

	//0x16d0b0a106e2de32b42ea4096c9e80c883c6ffa9e3f19f09cb45dfff2b02d09a3bcf95f2d0c33b7caf5db42d55d3459395c1b8d6a5d315a113edc39c4ce3a3d5269ab4a9514a998fdcc693d90a42505185270a184a07ddfb553b181be13e968480ef0df4c06cf657957b07118776a38fea3bcf758ea4491a4213719e2f6537b5
	//0x8e06b4ed3068274d276567dfddc7700a753d0fe407857e7720246ab0970e216273d3b4f66fe427b319ee62a9500e0507d6dd22c36e765448150a972d4ac049da77adbc919c2f73ea582fc49eba0d69b7cc5a215ebc1d94903e96d0ec356f1eda365cedf7275dabb1bf573b200a52e9b0e5d97861c25ea4cb4c8c6ada16e9f73b
	fmt.Println(mi.PubKey.GetHexString())

	//0x009f3b76f3e49dcdd6d2ee8421f077fd4c68c176b18e1e602a3c1f09f9272250
	//0x53ca357d46854058fe87fdcab0ba8e1b936521d60fd1c2ab1a6956d62fedd808
	fmt.Println(mi.VrfPK.GetHexString())

}
