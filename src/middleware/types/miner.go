package types

type Miner struct {
	Id           []byte
	PublicKey    []byte
	VrfPublicKey []byte

	Type  byte   // 提案者 还是验证者
	Stake uint64 // 质押数

	ApplyHeight uint64
	AbortHeight uint64

	Status byte // 当前状态
}
