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
	"com.tuntun.rangers/node/src/consensus/groupsig"
	"com.tuntun.rangers/node/src/consensus/vrf"
	"com.tuntun.rangers/node/src/middleware/types"
	"encoding/json"
	"fmt"
	"github.com/astaxie/beego/httplib"
)

type RemoteChainOpImpl struct {
	host string
	port int
	base string
	aop  accountOp
	show bool
}

func InitRemoteChainOp(ip string, port int, show bool, op accountOp) *RemoteChainOpImpl {
	ca := &RemoteChainOpImpl{
		aop:  op,
		show: show,
	}
	ca.Connect(ip, port)
	return ca
}

func (ca *RemoteChainOpImpl) Connect(ip string, port int) error {
	if ip == "" {
		return nil
	}
	ca.host = ip
	ca.port = port
	ca.base = fmt.Sprintf("http://%v:%v", ip, port)
	return nil
}

func (ca *RemoteChainOpImpl) request(method string, params ...interface{}) *Result {
	if ca.base == "" {
		return opError(ErrUnConnected)
	}
	req := httplib.Post(ca.base)

	param := RPCReqObj{
		Method:  "Rocket_" + method,
		Params:  params[:],
		ID:      1,
		Jsonrpc: "2.0",
	}

	if ca.show {
		fmt.Println("Request:")
		bs, _ := json.MarshalIndent(param, "", "\t")
		fmt.Println(string(bs))
		fmt.Println("==================================================================================")
	}

	req, err := req.JSONBody(param)
	if err != nil {
		return opError(err)
	}
	ret := &RPCResObj{}
	err = req.ToJSON(ret)

	if err != nil {
		return opError(err)
	}
	if ret.Error != nil {
		return opError(fmt.Errorf(ret.Error.Message))
	}
	return ret.Result
}

func (ca *RemoteChainOpImpl) nonce(addr string) (uint64, error) {
	ret := ca.request("nonce", addr)
	if !ret.IsSuccess() {
		return 0, fmt.Errorf(ret.Message)
	}
	return uint64(ret.Data.(float64)), nil
}

func (ca *RemoteChainOpImpl) Endpoint() string {
	return fmt.Sprintf("%v:%v", ca.host, ca.port)
}

func (ca *RemoteChainOpImpl) SendRaw(tx *txRawData) *Result {

	return nil
}

func (ca *RemoteChainOpImpl) Balance(addr string) *Result {
	return ca.request("balance", addr)
}

func (ca *RemoteChainOpImpl) MinerInfo(addr string) *Result {
	return ca.request("minerInfo", addr)
}

func (ca *RemoteChainOpImpl) BlockHeight() *Result {
	return ca.request("blockHeight")
}

func (ca *RemoteChainOpImpl) GroupHeight() *Result {
	return ca.request("groupHeight")
}

func (ca *RemoteChainOpImpl) TxInfo(hash string) *Result {
	return ca.request("transDetail", hash)
}

func (ca *RemoteChainOpImpl) BlockByHash(hash string) *Result {
	return ca.request("getBlockByHash", hash)
}

func (ca *RemoteChainOpImpl) BlockByHeight(h uint64) *Result {
	return ca.request("getBlockByHeight", h)
}

func (ca *RemoteChainOpImpl) ApplyMiner(mtype int, stake uint64, gas, gasprice uint64) *Result {
	r := ca.aop.AccountInfo()
	if !r.IsSuccess() {
		return r
	}
	aci := r.Data.(*Account)
	if aci.Miner == nil {
		return opError(fmt.Errorf("the current account is not a miner account"))
	}
	source := common.HexToAddress(aci.Address)
	var bpk groupsig.Pubkey
	bpk.SetHexString(aci.Miner.BPk)

	st := uint64(0)
	if mtype == common.MinerTypeValidator {
		fmt.Println("stake of applying verify node is hardened as 100 Tas")
		st = 1
	} else {
		st = stake * 1000000000
	}

	miner := &types.Miner{
		Id:           source.Bytes(),
		PublicKey:    bpk.Serialize(),
		VrfPublicKey: vrf.Hex2VRFPublicKey(aci.Miner.VrfPk).GetBytes(),
		Stake:        st,
		Type:         byte(mtype),
	}
	data, err := json.Marshal(miner)
	if err != nil {
		return opError(err)
	}

	tx := &txRawData{

		TxType: types.TransactionTypeMinerApply,
		Data:   common.ToHex(data),
	}
	ca.aop.(*AccountManager).resetExpireTime(aci.Address)
	return ca.SendRaw(tx)
}

func (ca *RemoteChainOpImpl) AbortMiner(mtype int, gas, gasprice uint64) *Result {
	r := ca.aop.AccountInfo()
	if !r.IsSuccess() {
		return r
	}
	aci := r.Data.(*Account)
	if aci.Miner == nil {
		return opError(fmt.Errorf("the current account is not a miner account"))
	}
	tx := &txRawData{
		TxType:    types.TransactionTypeMinerAbort,
		Data:      string([]byte{byte(mtype)}),
		ExtraData: aci.Address,
	}
	ca.aop.(*AccountManager).resetExpireTime(aci.Address)
	return ca.SendRaw(tx)
}

func (ca *RemoteChainOpImpl) RefundMiner(mtype int, gas, gasprice uint64) *Result {
	r := ca.aop.AccountInfo()
	if !r.IsSuccess() {
		return r
	}
	aci := r.Data.(*Account)
	if aci.Miner == nil {
		return opError(fmt.Errorf("the current account is not a miner account"))
	}
	tx := &txRawData{
		TxType:    types.TransactionTypeMinerRefund,
		Data:      string([]byte{byte(mtype)}),
		ExtraData: aci.Address,
	}
	ca.aop.(*AccountManager).resetExpireTime(aci.Address)
	return ca.SendRaw(tx)
}

func (ca *RemoteChainOpImpl) ViewContract(addr string) *Result {
	return ca.request("explorerAccount", addr)
}

func (ca *RemoteChainOpImpl) TxReceipt(hash string) *Result {
	return ca.request("txReceipt", hash)
}
