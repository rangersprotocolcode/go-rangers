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

package statemachine

import (
	"fmt"
	"os"
	"strings"

	"context"
	"github.com/ipfs/go-ipfs-api"
	"testing"
)

func TestGetLocalId(t *testing.T) {
	sh := shell.NewShell("localhost:5001")
	localID, err := sh.ID()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s", err)
		os.Exit(1)
	}
	fmt.Printf("local id:%v\n", localID.ID)
	addressList := localID.Addresses
	for i, address := range addressList {
		fmt.Printf("%d local address:%v\n", i, address)
	}

}

func TestAddSimpleFile(t *testing.T) {
	// Where your local node is running on localhost:5001
	sh := shell.NewShell("localhost:5001")
	file := strings.NewReader("hello world!")

	cid, err := sh.Add(file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s", err)
		os.Exit(1)
	}
	fmt.Printf("added %s", cid)
	//visit: https://ipfs.io/ipfs/{cid} to see the file content
}

func TestAddDockerImageFile(t *testing.T) {
	// Where your local node is running on localhost:5001
	sh := shell.NewShell("localhost:5001")
	file := "/home/x/docker/image/mixmarvel_image.tar"
	cid, err := sh.AddLink(file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s", err)
		os.Exit(1)
	}
	fmt.Printf("added %s", cid)
	//visit: https://ipfs.io/ipfs/{cid} to see the file content
}

func TestGetFile(t *testing.T) {
	// Where your local node is running on localhost:5001
	sh := shell.NewShell("localhost:5001")

	fileProviderAddress := "/ip4/47.110.143.114/tcp/4001/ipfs/QmWm1ZCAEHHPVxD411ZJk2qphQsEtLaPACGb3zrxtkPJuy"
	err := sh.SwarmConnect(context.Background(), fileProviderAddress)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s", err)
		os.Exit(1)
	}
	fmt.Print("connect ok!\n")

	//genesisImageId := "Qmeof3ShtuG5yt9jqx6xVV9RZjwVFb3PFddjyxaNg2YRMK"
	//snakeImageId:="QmXFNy9ADTR2Ljw7tTpWLPsWEFR21h8DbZBb1X9zdzBST2"
	mixmarvelImageId := "QmZCxfCbBkCYwYa5vt7YG3ctS19jVYSer3vc3f9b6pVJhk"
	err = sh.Get(mixmarvelImageId, "/Users/daijia/go/src/com.tuntun.rocket/node/src/statemachine/logs/mixmarvel_image.tar")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s", err)
		os.Exit(1)
	}
	fmt.Print("Got file!\n")
}

func TestGetLocalFile(t *testing.T) {
	sh := shell.NewShell("localhost:5001")

	fileProviderAddress := "/ip4/127.0.0.1/tcp/9001/ipfs/QmU8Pu6hkzJY1P4JmtgJCgy3Z52rqdChjvkysPmrqWGEkM"
	err := sh.SwarmConnect(context.Background(), fileProviderAddress)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s", err)
		os.Exit(1)
	}
	fmt.Print("connect ok!\n")

	//cid := "QmTp2hEo8eXRp6wg7jXv1BLCMh5a4F3B7buAUZNZUu772j"

}
