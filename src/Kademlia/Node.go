package Kademlia

import (
	"../../src/Chord"
	"math/big"
	"time"
)

const alpha = 3
const M = 160
const expireTime = time.Hour * 24

type Contact struct {
	IPAddr  string
	NodeNum *big.Int
}

type KVPair struct {
	key   string
	value string
}

type Node struct {
	bucket     []Kbucket
	self       Contact
	data       map[string]string
	expireTime map[string]time.Time //data automatically expires after expire time
}

func (this *Node) Ping(contact Contact) bool {
	client, err := this.Connect(contact)
	if err != nil {
		return false
	}
	var success bool
	err = client.Call("Node.Ping_", &this.self, &success)
	if err != nil {
		return false
	}
	return success
}

func (this *Node) FindNode(key *big.Int) []Contact {
	l := this.CalcPrefix(key)
	cur := this.bucket[l].Get(K, key)

}

func (this *Node) Put(key string, value string) bool {
	kClosest := this.FindNode(DHT.GetHash(key))
	expireTime := time.Now().Add(expireTime)
	for i := range kClosest {
		client, err := this.Connect(kClosest[i])
		if err != nil {
			continue
		}
		var reply StoreReturn
		err = client.Call("Node.Store", &StoreRequest{this.self, KVPair{key, value}, expireTime}, &reply)
		if err != nil || !reply.success || verifyIdentity(kClosest[i], reply.self) {
			return false
		}
	}
	return true
}
