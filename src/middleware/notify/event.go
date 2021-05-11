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

package notify

const (
	BlockAddSucc = "block_add_succ"

	GroupAddSucc = "group_add_succ"

	NewBlock = "new_block"

	TopBlockInfo = "top_block_info"

	BlockChainPieceReq = "block_chain_piece_req"

	BlockChainPiece = "block_chain_piece_info"

	BlockReq = "block_req"

	BlockResponse = "block_response"

	GroupChainPieceReq = "group_chain_piece_req"

	GroupChainPiece = "group_chain_piece_info"

	GroupReq = "group_req"

	GroupResponse = "group_response"

	TransactionReq = "transaction_req"

	TransactionGot = "transaction_got"

	TransactionGotAddSucc = "transaction_got_add_succ"

	ClientTransaction = "client_transaction"

	ClientTransactionRead = "client_transaction_read"

	CoinProxyNotify = "coin_proxy_notify"

	STMStorageReady = "STMStorageReady"

	AcceptGroup = "accept_group"

	WrongTxNonce = "wrongTxNonce"
)
