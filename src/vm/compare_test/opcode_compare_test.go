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

package compare_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
	"testing"
)

type OpCodeInfo struct {
	Code    string `json:"op"`
	GasCost uint64 `json:"gasCost"`
}

func (o OpCodeInfo) String() string {
	var buffer bytes.Buffer
	buffer.WriteString(o.Code)
	buffer.WriteString(":")
	buffer.WriteString(strconv.FormatUint(o.GasCost, 10))

	return buffer.String()
}

func TestOpcodeCompare(t *testing.T) {
	vmCodeList := getVMOPCode()
	ethCodeList := getETHOPCode()

	if len(vmCodeList) != len(ethCodeList) {
		fmt.Printf("Length not equal! %d-%d\n", len(vmCodeList), len(ethCodeList))
	}
	codeMap := make(map[string]string, 0)

	for i := 0; i < len(vmCodeList); i++ {
		vmCode := vmCodeList[i]
		ethCode := ethCodeList[i]
		fmt.Printf("%s---%s ", vmCode.String(), ethCode.String())

		if vmCode.Code == ethCode.Code && vmCode.GasCost == ethCode.GasCost {
			fmt.Print("ok\n")
			codeMap[vmCode.Code] = ""
		} else {
			fmt.Printf("Not Equal,index:%d\n", i)
			panic("Not Equal!!")
		}
	}

	instructionList := getAllInstructions()

	codeList := make([]string, 0)
	removedCodeList := make([]string, 0)
	for code, _ := range codeMap {
		codeList = append(codeList, code)
		if exits, index := contains(instructionList, code); exits {
			instructionList = append(instructionList[:index], instructionList[index+1:]...)
			removedCodeList = append(removedCodeList, code)
		}
	}
	fmt.Printf("Contain opcode:%v\n\n", codeList)
	fmt.Printf("Removed opcode:%v\n\n", removedCodeList)
	fmt.Printf("Final remain vm instrucitons:%d\n %v\n", len(instructionList), instructionList)

}

type ethJSON struct {
	Result  ethResult `json:"result"`
	Id      uint64    `json:"id"`
	Jsonrpc string    `json:"jsonrpc"`
}

type ethResult struct {
	Logs []OpCodeInfo `json:"structLogs"`
}

func getVMOPCode() []OpCodeInfo {
	opCodeList := make([]OpCodeInfo, 0)
	fileName := "vmlog.txt"

	bytes, err := ioutil.ReadFile(fileName)
	if err != nil {
		panic("read vm log info  file error:" + err.Error())
	}
	records := strings.Split(string(bytes), "\n")
	for _, record := range records {
		parts := strings.Split(record, "\"opName\":\"")
		parts = strings.Split(parts[1], "\",\"")

		opCodeInfo := OpCodeInfo{}
		opCodeInfo.Code = parts[0]

		parts = strings.Split(record, "\"gasCost\":")
		parts = strings.Split(parts[1], ",\"")
		opCodeInfo.GasCost, _ = strconv.ParseUint(parts[0], 10, 64)

		opCodeList = append(opCodeList, opCodeInfo)
	}
	return opCodeList
}

func getETHOPCode() []OpCodeInfo {
	fileName := "ethlog.json"

	bytes, err := ioutil.ReadFile(fileName)
	if err != nil {
		panic("read ethlog file error:" + err.Error())
	}

	var ethJson ethJSON
	err = json.Unmarshal(bytes, &ethJson)
	if err != nil {
		panic(err.Error())
	}
	return ethJson.Result.Logs
}

func getAllInstructions() []string {
	fileName := "instructions.txt"

	bytes, err := ioutil.ReadFile(fileName)
	if err != nil {
		panic("read instructions  file error:" + err.Error())
	}
	instructions := strings.Split(string(bytes), "\n")
	fmt.Printf("Got instructions len:%d\n %v\n\n", len(instructions), instructions)
	return instructions
}

func contains(array []string, str string) (bool, int) {
	for i := 0; i < len(array); i++ {
		if array[i] == str {
			return true, i
		}
	}
	return false, -1
}
