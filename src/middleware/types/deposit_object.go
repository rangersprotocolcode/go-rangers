package types

// 主链币
type DepositCoinData struct {
	ChainType        string `json:"chainType,omitempty"`
	Amount           string `json:"amount,omitempty"`
	TxId             string `json:"txId,omitempty"`
	MainChainAddress string `json:"Addr,omitempty"`
}

//FT充值确认数据结构
type DepositFTData struct {
	FTId             string `json:"ftId,omitempty"`
	Amount           string `json:"amount,omitempty"`
	MainChainAddress string `json:"Addr,omitempty"`
	ContractAddress  string `json:"ContractAddr,omitempty"`
	TxId             string `json:"txId,omitempty"`
}

//NFT充值确认数据结构
type DepositNFTData struct {
	SetId      string            `json:"setId,omitempty"`
	Name       string            `json:"name,omitempty"`
	Symbol     string            `json:"symbol,omitempty"`
	ID         string            `json:"id,omitempty"`
	Creator    string            `json:"creator,omitempty"`
	CreateTime string            `json:"createTime,omitempty"`
	Owner      string            `json:"owner,omitempty"`
	Renter     string            `json:"renter,omitempty"`
	Status     byte              `json:"status,omitempty"`
	Condition  byte              `json:"condition,omitempty"`
	AppId      string            `json:"appId,omitempty"`
	Data       map[string]string `json:"data,omitempty"`

	MainChainAddress string `json:"Addr,omitempty"`
	ContractAddress  string `json:"ContractAddr,omitempty"`
	TxId             string `json:"txId,omitempty"`
}

type Deposit struct {
	Method string            `json:"type,omitempty"`
	Data   map[string]string `json:"data,omitempty"`
}
