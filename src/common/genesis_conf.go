package common

type GenesisConf struct {
	Creator string `json:"creator"`

	ChainId string `json:"chainId"`
	Name    string `json:"name"`

	Cast       uint64 `json:"cast"`
	GroupLife  uint64 `json:"groupLife"`
	Proposals  uint64 `json:"p"`
	Validators uint64 `json:"v"`

	GenesisTime int64 `json:"genesisTime"`

	TimeCycle      int    `json:"timecycle"`
	TokenName      string `json:"tokenName"`
	TotalSupply    uint64 `json:"totalsupply"`
	Symbol         string `json:"symbol"`
	ReleaseRate    int    `json:"d"`
	ProposalToken  int    `json:"ptoken"`
	ValidatorToken int    `json:"vtoken"`

	// 生成的创始组与创始矿工
	Group        string   `json:"group"`
	JoinedGroup  string   `json:"joined"`
	ProposerInfo []string `json:"proposers"`

	Dev byte `json:"dev"`
}
