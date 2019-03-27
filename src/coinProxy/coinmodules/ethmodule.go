package coinmodules

import (
	"math/big"
	"github.com/ethereum/go-ethereum/core/types"
	//"github.com/stretchr/testify/require"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"crypto/ecdsa"
	"coinProxy/storestate"
	"github.com/ethereum/go-ethereum"
	"time"
)

type EthModule struct{
	KeyStoreFile	string

	FromPrivkey		*ecdsa.PrivateKey
	FromAddr		common.Address

	Client			*ethclient.Client
	Pstore			*storestate.StoreState
}

//func (self *EthModule)Depledge() error{
//
//	return nil
//}

func (self *EthModule)ThreadWatch(contractaddr string,url string) {
	client, err := ethclient.Dial(url)
	if err != nil {
		//log.Fatal(err)
		panic(err)
	}

	contractAddress := common.HexToAddress(contractaddr)
	query := ethereum.FilterQuery{
		Addresses: []common.Address{contractAddress},
	}

	logs := make(chan types.Log)
	sub, err := client.SubscribeFilterLogs(context.Background(), query, logs)
	if err != nil {
		//log.Fatal(err)
		panic(err)
	}

	for {
		select {
		case err := <-sub.Err():
			//log.Fatal(err)
			fmt.Println(err)
		case vLog := <-logs:
			fmt.Println(vLog) // pointer to event log
			var str string
			for _, value := range vLog.Topics{
				str = str + value.String()
			}
			self.Pstore.AddIncoming(vLog.Address.String(),vLog.TxHash.String(),common.Bytes2Hex(vLog.Data),str,*big.NewInt(int64(vLog.BlockNumber)))

			//self.Pstore.UpdateTransfered(vLog.TxHash.String(),*big.NewInt(int64(recp.GasUsed)),*big.NewInt(int64(recp.Logs[0].BlockNumber)),string(recp.Logs[0].Data),recp.Logs[0].Topics[0].String())
		}
	}


}

func (self *EthModule)ThreadBlockNumber(url string,count int) {
	client, err := ethclient.Dial(url)
	if err != nil {
		//log.Fatal(err)
		panic(err)
	}

	for {
		header, err := client.HeaderByNumber(context.Background(), nil)
		if err != nil {
			panic(err)
		}

		self.Pstore.UpdateFinished(*header.Number,count)

		time.Sleep(5*time.Second)
	}

}

func (self *EthModule)Call(contractaddr string,input []byte) error{
	go self.call(contractaddr,input)
	//time.Sleep(time.Second*20)
	return nil
}

func (self *EthModule)call(contractaddr string,input []byte) error{

	//fromKeystore,err := ioutil.ReadFile(self.KeyStoreFile)
	//require.NoError(t,err)
	//fromKey,err := keystore.DecryptKey(fromKeystore,password)
	//fromPrivkey := fromKey.PrivateKey
	//fromPubkey := fromPrivkey.PublicKey
	//fromAddr := crypto.PubkeyToAddress(fromPubkey)

	id,_:=self.Pstore.AddInfo("eth",self.FromAddr.String(),contractaddr,input)

	// 交易接收方
	toAddr := common.HexToAddress(contractaddr)

	// 数量
	tmp:=big.NewInt(100)
	amount := big.NewInt(1000000000000000000)
	amount.Mul(amount,tmp)
	//amount := big.Int{100000000000000000000}

	// gasLimit
	var gasLimit uint64 = 300000

	// 创建客户端
	//client, err:= ethclient.Dial(httpUrl)
	////require.NoError(t, err)
	//if err != nil{
	//	return fmt.Errorf("Dial %s error",httpUrl)
	//}
	var client *ethclient.Client = self.Client

	// gasPrice
	//gasPrice :=big.NewInt(10000)
	gasPrice,err := client.SuggestGasPrice(context.Background())
	if err != nil {
		return err
	}

	// nonce获取
	nonce, err := client.PendingNonceAt(context.Background(), self.FromAddr)

	// 认证信息组装
	auth := bind.NewKeyedTransactor(self.FromPrivkey)
	//auth,err := bind.NewTransactor(strings.NewReader(mykey),"111")
	auth.Nonce = big.NewInt(int64(nonce))
	auth.Value = amount     // in wei
	//auth.Value = big.NewInt(100000)     // in wei
	auth.GasLimit = gasLimit // in units
	//auth.GasLimit = uint64(0) // in units
	auth.GasPrice = gasPrice
	auth.From = self.FromAddr

	// 交易创建
	tx := types.NewTransaction(nonce,toAddr,amount,gasLimit,gasPrice,input)

	// 交易签名
	signedTx ,err:= auth.Signer(types.HomesteadSigner{}, auth.From, tx)
	//signedTx ,err := types.SignTx(tx,types.HomesteadSigner{},self.FromPrivkey)
	//require.NoError(t, err)
	if err != nil{
		return fmt.Errorf("auth.Signer error")
	}

	// 交易发送
	serr := client.SendTransaction(context.Background(),signedTx)
	if serr != nil {
		fmt.Println(serr)
		return serr
	}
	str:=string(signedTx.Hash().String())
	fmt.Println(str)

	self.Pstore.UpdatePending(id,*gasPrice,str)

	// 等待挖矿完成
	recp,erro := bind.WaitMined(context.Background(),client,signedTx)
	if erro != nil {
		return erro
	}
	if recp != nil{
		//fmt.Println(recp)
		fmt.Println("status:",recp.Status)
		fmt.Println("GasUsed:",recp.GasUsed)
		fmt.Println(recp.TxHash)
		fmt.Println(recp.Logs)
	}

	if recp.Status == 0 {
		self.Pstore.UpdateTransferedFailed(recp.TxHash.String(), *big.NewInt(int64(recp.GasUsed)))
	}else if recp.Status == 1{
		self.Pstore.UpdateTransfered(recp.TxHash.String(),*big.NewInt(int64(recp.GasUsed)),*big.NewInt(int64(recp.Logs[0].BlockNumber)),string(recp.Logs[0].Data),recp.Logs[0].Topics[0].String())
	}

	return nil
}