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

package service

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/middleware/types"
	"com.tuntun.rocket/node/src/storage/account"
	"com.tuntun.rocket/node/src/utility"
	"encoding/json"
	"fmt"
	"math/rand"
	"sort"
	"strconv"
	"strings"
)

type lotteryCondition struct {
	Combo  comboConditions  `json:"combo"`
	Prizes prizesConditions `json:"prizes"`
}

type prizesConditions struct {
	Nft nftConditions `json:"nft"`
	Ft  ftConditions  `json:"ft"`
}

type comboConditions struct {
	Probability string            `json:"p"`
	Content     map[string]string `json:"content"`
}

type nftConditions struct {
	Probability string            `json:"p"`
	Content     map[string]string `json:"content"`
}

type ftConditions struct {
	Probability string               `json:"p"`
	Content     map[string]ftContent `json:"content"`
}

type ftContent struct {
	Probability string `json:"p"`
	ValueRange  string `json:"range"`
}

type items struct {
	Nft []types.NFTID `json:"nft"`
	Ft  []types.FTID  `json:"ft"`
}

func CreateLottery(addr, condition string, accountDB *account.AccountDB) (string, string) {
	if 0 == len(condition) {
		txLogger.Error("CreateLottery error: condition empty")
		return "", "condition empty"
	}
	var lottery lotteryCondition
	err := json.Unmarshal(utility.StrToBytes(condition), &lottery)
	if nil != err {
		txLogger.Error("CreateLottery error: " + err.Error())
		return "", err.Error()
	}

	// check nft
	nft := lottery.Prizes.Nft
	if 0 == len(nft.Content) {
		probability, err := utility.StrToBigInt(nft.Probability)
		if nil != err {
			txLogger.Error("nft probability format error")
			return "", "nft probability format error"
		}
		if probability.Sign() > 0 {
			txLogger.Error("nft content is nil but p is not 0")
			return "", "nft content is nil but p is not 0"
		}
	}
	for setId := range nft.Content {
		nftSet := accountDB.GetNFTSetDefinition(setId)
		if nil == nftSet {
			txLogger.Errorf("nftSet: %s not existed", setId)
			return "", fmt.Sprintf("nftSet: %s not existed", setId)
		}
		if 0 != strings.Compare(nftSet.Owner, addr) {
			txLogger.Errorf("nftSet: %s owner error: %s vs %s", setId, nftSet.Owner, addr)
			return "", fmt.Sprintf("nftSet: %s owner error: %s vs %s", setId, nftSet.Owner, addr)
		}
		if 0 != nftSet.MaxSupply {
			txLogger.Errorf("nftSet: %s MaxSupply error, %d", setId, nftSet.MaxSupply)
			return "", fmt.Sprintf("nftSet: %s MaxSupply error, %d", setId, nftSet.MaxSupply)
		}
	}

	// check ft
	ft := lottery.Prizes.Ft
	if 0 == len(ft.Content) {
		probability, err := utility.StrToBigInt(ft.Probability)
		if nil != err {
			txLogger.Errorf("ft probability format error")
			return "", "ft probability format error"
		}
		if probability.Sign() > 0 {
			txLogger.Errorf("nft content is nil but p is not 0")
			return "", "nft content is nil but p is not 0"
		}
	}
	for ftId := range ft.Content {
		ftSet := FTManagerInstance.GetFTSet(ftId, accountDB)
		if nil == ftSet {
			txLogger.Errorf("ftSet: %s not existed", ftId)
			return "", fmt.Sprintf("ftSet: %s not existed", ftId)
		}
		if 0 != strings.Compare(ftSet.Owner, addr) {
			txLogger.Errorf("ftSet %s owner error, %s vs %s", ftId, ftSet.Owner, addr)
			return "", fmt.Sprintf("ftSet %s owner error, %s vs %s", ftId, ftSet.Owner, addr)
		}
		if 0 != ftSet.MaxSupply.Sign() {
			txLogger.Errorf("ftSet %s MaxSupply error, %s", ftId, ftSet.MaxSupply)
			return "", fmt.Sprintf("ftSet %s MaxSupply error, %s", ftId, ftSet.MaxSupply)
		}
	}

	nonce := accountDB.GetNonce(common.HexToAddress(addr))
	lotteryAddress := common.GenerateLotteryAddress(addr, nonce)
	accountDB.SetLotteryDefinition(lotteryAddress, utility.StrToBytes(condition), addr)

	txLogger.Debugf("addr: %s createLottery: %s with condition: %s", addr, lotteryAddress.GetHexString(), condition)
	return lotteryAddress.GetHexString(), ""
}

func Jackpot(lotteryAddress, target, time string, seed, height uint64, accountDB *account.AccountDB) (string, string) {
	address := common.HexToAddress(lotteryAddress)
	targetAddress := common.HexToAddress(target)

	condition := accountDB.GetLotteryDefinition(address)
	if 0 == len(condition) {
		txLogger.Errorf("no such lotteryAddress: %s", lotteryAddress)
		return "", fmt.Sprintf("no such lotteryAddress: %s", lotteryAddress)
	}

	//pre := utility.ByteToUInt64(accountDB.GetData(address, utility.StrToBytes(target)))
	//if 0 != pre && height-pre < common.BlocksPerDay {
	//	txLogger.Errorf("not the time. height: %d, pre: %d, lottery: %s, user: %s", height, pre, lotteryAddress, target)
	//	return "", fmt.Sprintf("not the time. height: %d, pre: %d, lottery: %s, user: %s", height, pre, lotteryAddress, target)
	//}

	var lottery lotteryCondition
	err := json.Unmarshal(condition, &lottery)
	if nil != err {
		txLogger.Error(err.Error())
		return "", err.Error()
	}

	comboP, err := strconv.ParseFloat(lottery.Combo.Probability, 64)
	if nil != err {
		txLogger.Error(err.Error())
		return "", err.Error()
	}
	random := rand.New(rand.NewSource(int64(seed)))

	num := 1
	// 计算能抽几次
	if comboP != 0 {
		p := random.Float64()
		if comboP == 1 || p > comboP {
			q := random.Float64()
			item := getJackPotItem(q, lottery.Combo.Content)
			if 0 != len(item) {
				num, err = strconv.Atoi(item)
				if nil != err {
					txLogger.Error(err.Error())
					return "", err.Error()
				}
			}

		}
	}

	award := items{Nft: make([]types.NFTID, 0), Ft: make([]types.FTID, 0)}

	// 开始抽奖
	owner := accountDB.GetLotteryOwner(address)
	nftProbability, err := strconv.ParseFloat(lottery.Prizes.Nft.Probability, 64)
	if nil != err {
		txLogger.Error(err.Error())
		return "", err.Error()
	}
	ftProbability, err := strconv.ParseFloat(lottery.Prizes.Ft.Probability, 64)
	if nil != err {
		txLogger.Error(err.Error())
		return "", err.Error()
	}
	for i := 0; i < num; i++ {
		p := random.Float64()

		if p < nftProbability {
			// 抽中nft
			p = random.Float64()
			nftSetId := getJackPotItem(p, lottery.Prizes.Nft.Content)
			if 0 == len(nftSetId) {
				continue
			}

			nonce := accountDB.GetNonce(common.GenerateNFTSetAddress(nftSetId))
			id := strconv.FormatUint(nonce, 10)
			NFTManagerInstance.MintNFT(owner, owner, nftSetId, id, "", time, targetAddress, accountDB)
			award.Nft = append(award.Nft, types.NFTID{SetId: nftSetId, Id: id})
		} else if p < (nftProbability + ftProbability) {
			// 抽中ft
			p = random.Float64()
			ftSetId, value := getJackPotFt(p, lottery.Prizes.Ft.Content, random)
			if 0 == len(ftSetId) {
				continue
			}

			FTManagerInstance.MintFT(owner, ftSetId, target, value, accountDB)
			award.Ft = append(award.Ft, types.FTID{Id: ftSetId, Value: value})
		}
	}

	accountDB.SetData(address, utility.StrToBytes(target), utility.UInt64ToByte(height))
	data, _ := json.Marshal(award)
	awardString := utility.BytesToStr(data)
	txLogger.Debugf("addr: %s get award: %s from lottery: %s", target, awardString, lotteryAddress)
	return awardString, ""
}

func getJackPotFt(p float64, prizes map[string]ftContent, random *rand.Rand) (string, string) {
	sortedKeys := make([]string, 0)
	for k, _ := range prizes {
		sortedKeys = append(sortedKeys, k)
	}
	sort.Strings(sortedKeys)

	sum := float64(0)
	for _, key := range sortedKeys {
		ftContent := prizes[key]
		probability, err := strconv.ParseFloat(ftContent.Probability, 64)
		if nil != err {
			return "", ""
		}
		sum += probability

		if p < sum {
			bounders := strings.Split(ftContent.ValueRange, "-")
			if 2 != len(bounders) {
				return "", ""
			}
			low, err := strconv.ParseFloat(strings.TrimSpace(bounders[0]), 64)
			if err != nil {
				return "", ""
			}
			high, err := strconv.ParseFloat(strings.TrimSpace(bounders[1]), 64)
			if err != nil {
				return "", ""
			}

			coins := low + random.Float64()*(high-low)
			return key, strconv.FormatFloat(coins, 'f', 8, 64)
		}
	}

	return "", ""
}

func getJackPotItem(p float64, prizes map[string]string) string {
	sortedKeys := make([]string, 0)
	for k, _ := range prizes {
		sortedKeys = append(sortedKeys, k)
	}
	sort.Strings(sortedKeys)

	sum := float64(0)
	for _, key := range sortedKeys {
		value := prizes[key]
		probability, err := strconv.ParseFloat(value, 64)
		if nil != err {
			return ""
		}
		sum += probability

		if p < sum {
			return key
		}
	}

	return ""
}
