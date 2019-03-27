package beCall

import (
	"coinProxy/coinmodules"
	"fmt"
	"github.com/ethereum/go-ethereum/crypto"
	"io/ioutil"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/ethclient"
	"coinProxy/storestate"
)

type CallModule struct{
	Store storestate.StoreState

	modeth	coinmodules.EthModule
	modont	coinmodules.OntModule
}

// http服务地址, 例:http://localhost:8545
//var httpUrl = "http://127.0.0.1:8545"
var httpUrl = "ws://127.0.0.1:8546"

// keystore文件对应的密码
var password = "" //"Password1qaz@WSX";

//初始化，设置各币种付款账号
func (self *CallModule)Init(gasaccounts map[string]string) error{
	if(len(gasaccounts["eth"]) != 0){
		self.modeth.Pstore = &self.Store

		self.modeth.KeyStoreFile = gasaccounts["eth"]

		fromKeystore,_ := ioutil.ReadFile(self.modeth.KeyStoreFile)
		//require.NoError(t,err)
		fromKey,_ := keystore.DecryptKey(fromKeystore,password)
		fromPrivkey := fromKey.PrivateKey
		fromPubkey := fromPrivkey.PublicKey
		fromAddr := crypto.PubkeyToAddress(fromPubkey)

		self.modeth.FromPrivkey = fromPrivkey
		self.modeth.FromAddr = fromAddr

		// 创建客户端
		var err error
		self.modeth.Client,err = ethclient.Dial(httpUrl)
		//require.NoError(t, err)
		if err != nil{
			return fmt.Errorf("Dial %s error",httpUrl)
		}
	}
	if(len(gasaccounts["ont"]) != 0){
		self.modont.KeyStoreFile = gasaccounts["ont"]
	}

	//init db
	self.Store.Init()

	//create thread to filter all contract's receipts
	addresss,_ := self.Store.GetAllGames()
	for k,v := range addresss {
		fmt.Println(k,v)
		if v == "eth" {
			go self.modeth.ThreadWatch(k,"ws://127.0.0.1:8546")
			go self.modeth.ThreadBlockNumber("ws://127.0.0.1:8546",8)
		}else if v == "ont" {
			//???self.modont.
		}
	}

	return nil
}

func (self *CallModule)DeInit() error {
	return self.Store.Deinit()
}

//提现
//func (self *CallModule)Depledge(cointype string,) error{
//	if cointype=="eth" {
//		return self.modeth.Depledge()
//	}else if cointype=="ont" {
//		//???return self.modont.Depledge()
//	}
//	return fmt.Errorf("Depledge , coin type error")
//}

func (self *CallModule)AddGame(cointype string,gamename string,contractaddress string) error {
	err :=self.Store.AddGame(cointype,gamename,contractaddress)
	if err != nil {
		return err
	}
	//???create thread
	return nil
}

func (self *CallModule)DelGame(cointype string,contractaddress string) error {
	return self.Store.DelGame(cointype,contractaddress)
}

//游戏的调用链接口
func (self *CallModule)Call(cointype string,contractaddr string,input []byte) error{
	if cointype=="eth" {
		return self.modeth.Call(contractaddr,input)
	}else if cointype=="ont" {
		//???return self.modont.Depledge()
	}
	return fmt.Errorf("Call , coin type error")
}

