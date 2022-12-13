package common

type GenesisConf struct {
	ChainId string `json:"chainId"`
	Name    string `json:"name"`

	Cast      uint64 `json:"cast"`
	GroupLife uint64 `json:"groupLife"`

	GenesisTime int64 `json:"genesisTime"`

	Group       string `json:"group"`
	JoinedGroup string `json:"joined"`
}
