package eth_rpc

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/eth_tx"
	"com.tuntun.rocket/node/src/middleware/types"
	"com.tuntun.rocket/node/src/storage/rlp"
	"com.tuntun.rocket/node/src/utility"
)

type ethAPIService struct{}

// SendRawTransaction will add the signed transaction to the transaction pool.
// The sender is responsible for signing the transaction and using the correct nonce.
func (api *ethAPIService) SendRawTransaction(encodedTx utility.Bytes) (common.Hash, *types.Transaction, error) {
	tx := new(eth_tx.Transaction)
	if err := rlp.DecodeBytes(encodedTx, tx); err != nil {
		return common.Hash{}, nil, err
	}
	logger.Debugf("raw tx hash:%v", tx.Hash().String())

	signer := eth_tx.NewEIP155Signer(common.GetChainId(true))
	sender, err := eth_tx.Sender(signer, tx)
	if err != nil {
		return common.Hash{}, nil, err
	}

	rocketTx := eth_tx.ConvertTx(tx, sender, encodedTx)
	return rocketTx.Hash, rocketTx, nil
}
