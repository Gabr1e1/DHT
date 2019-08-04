package Kademlia

import (
	"../../src/Chord"
	"fmt"
	"log"
	"math/big"
	"net"
	"net/rpc"
	"sync"
	"time"
)

const alpha = 3
const M = 160
const expireTime = time.Hour * 24
const checkInterval = time.Hour
const republishInterval = time.Hour

type Contact struct {
	Id *big.Int
	Ip string
}

type KVPair struct {
	Key string
	Val string
}

type Set map[string]struct{} //F**k, don't want to use this, but somehow have to

type Node struct {
	bucket     []KBucket
	Self       Contact
	data       map[string]Set
	expireTime map[KVPair]time.Time //Pair automatically expires after Expire time
	republish  map[KVPair]bool      //whether it needs to be republished this hour

	dataMux sync.RWMutex

	server   *rpc.Server
	listener net.Listener
}

func (this *Node) GetContact(_ *int, reply *Contact) error {
	*reply = this.Self
	return nil
}

func (this *Node) Create(addr string) {
	this.Self = Contact{DHT.GetHash(addr), addr}
	for i := 0; i <= M; i++ {
		this.bucket = append(this.bucket, KBucket{})
	}
	this.data = make(map[string]Set)
	this.expireTime = make(map[KVPair]time.Time)
	this.republish = make(map[KVPair]bool)

	fmt.Println("Node Created", this.Self)
}

func (this *Node) Run() {
	this.server = rpc.NewServer()
	_ = this.server.Register(this)

	var err error = nil
	this.listener, err = net.Listen("tcp", this.Self.Ip)
	if err != nil {
		log.Fatal("listen error: ", err)
	}
	go this.server.Accept(this.listener)
	go this.expireCheck()
}

func (this *Node) Join(addr string) {
	other := Contact{DHT.GetHash(addr), addr}

	//insert node into the appropriate bucket list
	l := this.CalcPrefix(other.Id)
	this.bucket[l].insert(this, other)

	//Call FindNode on itself
	this.FindNode(this.Self.Id)
}

func (this *Node) Ping(contact Contact) bool {
	client, err := this.Connect(contact)
	if err != nil {
		fmt.Println("Connection failed", err)
		return false
	}
	var ret PingReturn
	err = client.Call("Node.RPCPing", &this.Self, &ret)
	_ = client.Close()
	if err != nil {
		fmt.Println("Ping failed", err)
		return false
	}
	return ret.Success
}

func (this *Node) GetClosest(key *big.Int, num int) []Contact {
	l := this.CalcPrefix(key)
	cur := this.bucket[l].Get(num)
	if len(cur) < num {
		var t []Contact
		for i := 0; i < len(this.bucket); i++ {
			if i == l {
				continue
			}
			t = append(t, this.bucket[i].contacts...)
		}
		cur = append(cur, GetClosestInList(key, num-len(cur), t)...)
	}
	return cur
}

func (this *Node) FindNode(hashId *big.Int) []Contact {
	cur := this.GetClosest(hashId, alpha)
	var ans map[string]Contact
	ans = make(map[string]Contact)

	for len(cur) > 0 {
		contact := cur[0]
		cur = cur[1:]

		client, err := this.Connect(contact)
		if err != nil {
			fmt.Println(this.Self, "Can't call", contact, err)
			continue
		}
		var reply FindNodeReturn
		err = client.Call("Node.RPCFindNode", &FindNodeRequest{this.Self, hashId}, &reply)
		_ = client.Close()
		if err != nil {
			fmt.Println(this.Self, "Can't call RPCFindNode in", contact)
			continue
		}
		t := reply.Closest
		for _, i := range t {
			_, ok := ans[i.Ip]
			if ok {
				continue
			}
			ans[i.Ip] = i
			cur = append(cur, i)
		}
	}
	var ansList []Contact
	for _, v := range ans {
		ansList = append(ansList, v)
	}
	//fmt.Println(ansList)
	return GetClosestInList(hashId, K, ansList)
}

func (this *Node) FindValue(hashId *big.Int, key string) Set {
	//Try to find value in itself
	if _, ok := this.data[key]; ok {
		return this.data[key]
	}

	cur := this.GetClosest(hashId, alpha)
	ans := make(map[string]Contact)
	for len(cur) > 0 {
		//fmt.Println(len(cur), cur)
		contact := cur[0]
		cur = cur[1:]

		client, err := this.Connect(contact)
		if err != nil {
			fmt.Println(this.Self, "Can't call", contact)
			continue
		}
		var reply FindValueReturn
		err = client.Call("Node.RPCFindValue", &FindValueRequest{this.Self, hashId, key}, &reply)
		_ = client.Close()
		if err != nil {
			fmt.Println(this.Self, "Can't call FindValue in", contact, err)
			continue
		}

		if reply.Val != nil {
			return reply.Val
		}

		t := reply.Closest
		for _, i := range t {
			_, ok := ans[i.Ip]
			if ok {
				continue
			}
			ans[i.Ip] = i
			cur = append(cur, i)
		}
	}
	return nil
}

func (this *Node) Put(key string, value string) bool {
	kClosest := this.FindNode(DHT.GetHash(key))
	//fmt.Println("PUT", key, kClosest)
	expireTime := time.Now().Add(expireTime)
	for i := range kClosest {
		client, err := this.Connect(kClosest[i])
		if err != nil {
			continue
		}
		var reply StoreReturn
		err = client.Call("Node.RPCStore", &StoreRequest{this.Self, KVPair{key, value}, expireTime}, &reply)
		_ = client.Close()
		if err != nil || !reply.Success || !verifyIdentity(kClosest[i], reply.Header) {
			return false
		}
	}
	return true
}

func (this *Node) Replicate(key string, value string) bool {
	kClosest := this.FindNode(DHT.GetHash(key))
	//use existing expire time
	expireTime := this.expireTime[KVPair{key, value}]

	for i := range kClosest {
		client, err := this.Connect(kClosest[i])
		if err != nil {
			continue
		}
		var reply StoreReturn
		err = client.Call("Node.RPCStore", &StoreRequest{this.Self, KVPair{key, value}, expireTime}, &reply)
		_ = client.Close()
		if err != nil || !reply.Success || !verifyIdentity(kClosest[i], reply.Header) {
			return false
		}
	}
	return true
}

func (this *Node) Get(key string) (bool, []string) {
	val := this.FindValue(DHT.GetHash(key), key)
	ret := make([]string, 0)
	for k := range val {
		ret = append(ret, k)
	}
	return val != nil, ret
}

func (this *Node) expireCheck() {
	for {
		this.check()
		time.Sleep(checkInterval)
	}
}

func (this *Node) check() {
	this.dataMux.RLock()
	for key, set := range this.data {
		for value := range set {
			if this.expireTime[KVPair{key, value}].Before(time.Now()) {
				this.dataMux.RUnlock()
				this.dataMux.Lock()

				delete(this.data, key)
				delete(this.expireTime, KVPair{key, value})

				this.dataMux.Unlock()
				this.dataMux.RLock()
			}
		}
	}
	this.dataMux.RUnlock()

}

func (this *Node) replicateKey() {
	for {
		for key, set := range this.data {
			for value := range set {
				_, ok := this.republish[KVPair{key, value}]
				if ok && this.republish[KVPair{key, value}] == false {
					this.republish[KVPair{key, value}] = true
					continue
					this.Replicate(key, value)
				}
			}
		}
		time.Sleep(republishInterval)
	}
}
