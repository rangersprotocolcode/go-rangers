package core

import (
	"testing"
	"fmt"
	"encoding/json"
	"x/src/statemachine"
	"x/src/middleware/types"
)

func TestStartStateMachineTx(t *testing.T) {
	containerConfig := statemachine.ContainerConfig{Priority: 0, Game: "0x0b7467fe7225e8adcb6b5779d68c20fceaa58d54",
		Image: "littlebear234/genesis_image:latest", Detached: true, Hostname: "genesis_host_name"}

	port := statemachine.Port{Host: 0, Target: 0}
	ports := statemachine.Ports{port}
	containerConfig.Ports = ports

	containerConfig.DownloadUrl = "littlebear234/genesis_image:latest"
	containerConfig.DownloadProtocol = "pull"

	tx := types.Transaction{Source: "0x0b7467fe7225e8adcb6b5779d68c20fceaa58d54", Target: "", Type: types.TransactionTypeAddStateMachine, Time: "12121"}
	tx.Data = containerConfig.TOJSONString()

	tx.Hash = tx.GenHash()

	j, _ := json.Marshal(tx.ToTxJson())
	fmt.Printf("TX JSON:\n%s\n", string(j))
}
