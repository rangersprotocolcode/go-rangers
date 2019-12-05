package statemachine

import (
	"fmt"
	"strings"
	"os"

	"github.com/ipfs/go-ipfs-api"
	"testing"
	"context"
)

func TestGetLocalId(t *testing.T) {
	sh := shell.NewShell("localhost:5001")
	localID, err := sh.ID()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s", err)
		os.Exit(1)
	}
	fmt.Printf("local id:%v\n", localID)
	addressList := localID.Addresses
	fmt.Printf("local address:%v\n", addressList[len(addressList)-1])
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
	err = sh.Get(mixmarvelImageId, "/home/ipfs/mixmarvel_image.tar")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s", err)
		os.Exit(1)
	}
	fmt.Print("Got file!\n")
}
