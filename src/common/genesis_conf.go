package common

type GenesisConf struct {
	ChainId string `json:"chainId"`
	Name    string `json:"name"`

	Cast      int `json:"cast"`
	GroupLife int `json:"groupLife"`

	GenesisTime int64 `json:"genesisTime"`

	Group       string `json:"group"`
	JoinedGroup string `json:"joined"`
}
