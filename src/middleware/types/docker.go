package types

type Nonce struct {
	Nonce int `json:"nonce"`
}

type PayloadMessage struct {
	Timestamp int    `json:"timestamp"`
	MsgName   string `json:"msg_name"`
	MsgData   string `json:"msg_data"`
}

type OutputMessage struct {
	Status int        `json:"status"`
	Data   OutputData `json:"data"`
}

type OutputData struct {
	LastNonce int `json:"last_nonce"`
	ErrCode   int `json:"err_code"`
}
