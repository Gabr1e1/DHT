package DHT

import (
	"log"
	"net/rpc"
)

func (n *Node) connect(otherNode InfoType) *rpc.Client {
	client, err := rpc.Dial("tcp", otherNode.IPAddr)
	if err != nil {
		log.Fatal("Connection Error: ", err)
		return nil
	}
	return client
}
