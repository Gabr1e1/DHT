package Kademlia

import "time"

func (this *Node) update(contact Contact) {
	belong := this.CalcPrefix(contact.NodeNum)
	this.bucket[belong].insert(this, contact)
}

type PingReturn struct {
	success bool
	self    Contact
}

func (this *Node) Ping_(sender *Contact, reply *PingReturn) error {
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

func (this *Node) Store(request *StoreRequest, reply *StoreReturn) error {
	this.update(request.sender)
	this.data[request.data.key] = request.data.value
	this.expireTime[request.data.key] = request.expire
	*reply = StoreReturn{true, this.self}
	return nil
}
