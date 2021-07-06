package eth_rpc

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/middleware/types"
	"com.tuntun.rocket/node/src/storage/rlp"
	"com.tuntun.rocket/node/src/utility"
)

type ethAPIService struct{}

// SendRawTransaction will add the signed transaction to the transaction pool.
// The sender is responsible for signing the transaction and using the correct nonce.
func (api *ethAPIService) SendRawTransaction(encodedTx utility.Bytes) (common.Hash, *types.Transaction, error) {
	tx := new(Transaction)
	if err := rlp.DecodeBytes(encodedTx, tx); err != nil {
		return common.Hash{}, nil, err
	}
	logger.Debugf("raw tx hash:%v", tx.Hash().String())

	signer := NewEIP155Signer(common.GetChainId())
	sender, err := Sender(signer, tx)
	if err != nil {
		return common.Hash{}, nil, err
	}

	rocketTx := convertTx(tx, sender)
	return rocketTx.Hash, rocketTx, nil
}
