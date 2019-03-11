package network

type Network interface {
	//Send message to the node which id represents.If self doesn't connect to the node,
	// resolve the kad net to find the node and then send the message
	Send(id string, msg Message) error

	//Send message to the node which id represents. If self doesn't connect to the node,
	// send message to the guys which belongs to the same group with the node and they will rely the message to the node
	SendWithGroupRelay(id string, groupId string, msg Message) error

	//Random broadcast the message to parts nodes in the group which self belongs to
	RandomSpreadInGroup(groupId string, msg Message) error

	//Broadcast the message among the group which self belongs to
	SpreadAmongGroup(groupId string, msg Message) error

	//send message to random memebers which in special group
	SpreadToRandomGroupMember(groupId string, groupMembers []string, msg Message) error

	//Broadcast the message to the group which self do not belong to
	//SpreadToGroup(groupId string, groupMembers []string, msg Message, digest MsgDigest) error

	//Send message to neighbor nodes
	TransmitToNeighbor(msg Message) error

	//Send the message to part nodes it connects to and they will also broadcast the message to part of their neighbor util relayCount
	Relay(msg Message, relayCount int32) error

	//Send the message to all nodes it connects to and the node which receive the message also broadcast the message to their neighbor once
	Broadcast(msg Message) error

	//Return all connections self has
	//ConnInfo() []Conn

	BuildGroupNet(groupId string, members []string)

	DissolveGroupNet(groupId string)
}
