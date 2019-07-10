package main

import (
	"log"
	"net/rpc"
)

func (n *Node) connect(otherNode *Node) *rpc.Client {
	client, err := rpc.DialHTTP("tcp", otherNode.IPAddr)
	if err != nil {
		log.Fatal("Connection Error: ", err)
		return nil
	}
	return client
}
