//TODO: Implement republish

package Kademlia

import (
	"math/big"
	"time"
)

func (this *Node) update(contact Contact) {
	if contact.Ip == this.Self.Ip {
		return
	}
	belong := this.CalcPrefix(contact.Id)
	this.bucket[belong].insert(this, contact)
}

type PingReturn struct {
	Header  Contact
	Success bool
}

func (this *Node) RPCPing(sender *Contact, reply *PingReturn) error {
	go this.update(*sender)

	*reply = PingReturn{this.Self, true}
	return nil
}

type StoreRequest struct {
	Header Contact
	Data   KVPair
	Expire time.Time
}

type StoreReturn struct {
	Header  Contact
	Success bool
}

func (this *Node) RPCStore(request *StoreRequest, reply *StoreReturn) error {
	go this.update(request.Header)

	this.data[request.Data.Key] = request.Data.Val
	this.expireTime[request.Data.Key] = request.Expire
	this.republish[request.Data.Key] = false

	*reply = StoreReturn{this.Self, true}
	return nil
}

type FindNodeRequest struct {
	Header Contact
	Id     *big.Int
}

type FindNodeReturn struct {
	Header  Contact
	Closest []Contact
}

func (this *Node) RPCFindNode(request *FindNodeRequest, reply *FindNodeReturn) error {
	go this.update(request.Header)
	cur := this.GetClosest(request.Id, K)
	*reply = FindNodeReturn{this.Self, cur}
	return nil
}

type FindValueRequest struct {
	Header Contact
	Id     *big.Int
	Key    string
}

type FindValueReturn struct {
	Closest []Contact
	Self    Contact
	Val     string
}

func (this *Node) RPCFindValue(request *FindValueRequest, reply *FindValueReturn) error {
	go this.update(request.Header)

	if _, ok := this.data[request.Key]; ok {
		*reply = FindValueReturn{nil, this.Self, this.data[request.Key]}
		return nil
	}
	cur := this.GetClosest(request.Id, K)
	*reply = FindValueReturn{cur, this.Self, ""}
	return nil
}
