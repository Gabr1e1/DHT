package Kademlia

import (
	"math/big"
	"time"
)

func (this *Node) update(contact Contact) {
	belong := this.CalcPrefix(contact.NodeNum)
	this.bucket[belong].insert(this, contact)
}

type PingReturn struct {
	success bool
	self    Contact
}

func (this *Node) RPCPing(sender *Contact, reply *PingReturn) error {
	this.update(*sender)
	*reply = PingReturn{true, this.self}
	return nil
}

type StoreRequest struct {
	sender Contact
	data   KVPair
	expire time.Time
}

type StoreReturn struct {
	success bool
	self    Contact
}

func (this *Node) RPCStore(request *StoreRequest, reply *StoreReturn) error {
	this.update(request.sender)
	this.data[request.data.key] = request.data.value
	this.expireTime[request.data.key] = request.expire
	*reply = StoreReturn{true, this.self}
	return nil
}

type FindNodeRequest struct {
	sender Contact
	id     *big.Int
}

type FindNodeReturn struct {
	closest []Contact
	self    Contact
}

func (this *Node) RPCFindNode(request *FindNodeRequest, reply *FindNodeReturn) error {
	cur := this.GetClosest(request.id, K)
	*reply = FindNodeReturn{cur, this.self}
	return nil
}

type FindValueRequest = FindNodeRequest

//type Set = map[string]struct{}

type FindValueReturn struct {
	closest []Contact
	self    Contact
	val     string
}

func (this *Node) RPCFindValue(request *FindValueRequest, reply *FindValueReturn) error {
	if _, ok := this.data[request.id.String()]; ok {
		*reply = FindValueReturn{nil, this.self, this.data[request.id.String()]}
		return nil
	}
	var tmp FindNodeReturn
	err := this.RPCFindNode(request, &tmp)
	*reply = FindValueReturn{tmp.closest, this.self, ""}
	return err
}
