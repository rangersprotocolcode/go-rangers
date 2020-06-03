package common

const (
	EPOCH               int = 5
	Group_Create_Gap        = 50
	GROUP_Work_GAP          = Group_Create_Gap + EPOCH*8                  //组准备就绪后, 等待可以铸块的间隔为4个epoch
	GROUP_Work_DURATION     = 2 * 60 * 60 * 1000 / CastingInterval //组铸块的周期为100个epoch
)