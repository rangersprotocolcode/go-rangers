package types

type Nonce struct {
	Nonce int `json:"nonce"`
}


type OutputMessage struct {
	Status int        `json:"status"`
	Payload   string `json:"payload"`
}

