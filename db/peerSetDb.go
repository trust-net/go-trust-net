package db

// Interface for a Node
type PeerNode interface {
	AddTx(tx interface{})
	HasTx(tx interface{}) bool
	Id() string
}

//	Interface to implement a lookup that can help find nodes that have seen a message
type PeerSetDb interface {
	// return the node for given peer
	PeerNodeForId(id string) PeerNode

	// return a list of nodes that have not seen the message
	PeerNodesWithMsgNotSeen(item interface{}) []PeerNode

	// register a node with lookup implementation
	RegisterPeerNode(node PeerNode) error
	
	// remove node entry for the peer from lookup implementation
	UnRegisterPeerNodeForId(id string) error
	
	// check if a message has been seen by me before
	HaveISeenIt(item interface{}) bool
}

