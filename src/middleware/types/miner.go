package types

type Miner struct {
	Id           []byte `json:"id,omitempty"`
	PublicKey    []byte `json:"publicKey,omitempty"`
	VrfPublicKey []byte `json:"vrfPublicKey,omitempty"`

	// 提案者 还是验证者
	Type byte `json:"type,omitempty"`

	// 质押数
	Stake uint64 `json:"stake,omitempty"`

	ApplyHeight uint64 `json:"-"`
	AbortHeight uint64 `json:"-"`

	Status byte // 当前状态
}
