package Kademlia

import (
	"math/big"
	"time"
)

func (this *Node) update(contact Contact) {
	if contact.IPAddr == this.Self.IPAddr {
		return
	}
	belong := this.CalcPrefix(contact.NodeNum)
	this.bucket[belong].insert(this, contact)
}

type PingReturn struct {
	Success bool
	Self    Contact
}

func (this *Node) RPCPing(sender *Contact, reply *PingReturn) error {
	go this.update(*sender)

	*reply = PingReturn{true, this.Self}
	return nil
}

type StoreRequest struct {
	Sender Contact
	Data   KVPair
	Expire time.Time
}

type StoreReturn struct {
	Success bool
	Self    Contact
}

func (this *Node) RPCStore(request *StoreRequest, reply *StoreReturn) error {
	go this.update(request.Sender)

	this.data[request.Data.Key] = request.Data.Value
	this.expireTime[request.Data.Key] = request.Expire
	this.republish[request.Data.Key] = false

	*reply = StoreReturn{true, this.Self}
	return nil
}

type FindNodeRequest struct {
	Sender Contact
	Id     *big.Int
}

type FindNodeReturn struct {
	Closest []Contact
	Self    Contact
}

func (this *Node) RPCFindNode(request *FindNodeRequest, reply *FindNodeReturn) error {
	go this.update(request.Sender)
	cur := this.GetClosest(request.Id, K)
	*reply = FindNodeReturn{cur, this.Self}
	return nil
}

type FindValueRequest struct {
	Sender Contact
	Id     *big.Int
	Key    string
}

type FindValueReturn struct {
	Closest []Contact
	Self    Contact
	Val     string
}

func (this *Node) RPCFindValue(request *FindValueRequest, reply *FindValueReturn) error {
	go this.update(request.Sender)

	if _, ok := this.data[request.Key]; ok {
		*reply = FindValueReturn{nil, this.Self, this.data[request.Key]}
		return nil
	}
	cur := this.GetClosest(request.Id, K)
	*reply = FindValueReturn{cur, this.Self, ""}
	return nil
}
