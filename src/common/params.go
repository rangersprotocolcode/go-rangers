package common

var (
	isFullNode = false
)

func IsFullNode() bool {
	return isFullNode
}

func SetFullNode(fullNode bool) {
	isFullNode = fullNode
}
