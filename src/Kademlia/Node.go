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

type Node struct {
	bucket     []KBucket
	Self       Contact
	data       map[string]string
	expireTime map[string]time.Time //Data automatically expires after Expire time
	republish  map[string]bool      //whether it needs to be republished this hour

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
	this.data = make(map[string]string)
	this.expireTime = make(map[string]time.Time)
	this.republish = make(map[string]bool)

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
	//get node info using addr
	client, err := this.Connect(Contact{nil, addr})
	if err != nil {
		fmt.Println(this.Self, "Can't Join", addr)
		return
	}
	var other Contact
	err = client.Call("Node.GetContact", 0, &other)
	_ = client.Close()
	if err != nil {
		fmt.Println(this.Self, "Can't get contact info", addr)
		return
	}

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

func (this *Node) FindValue(hashId *big.Int, key string) string {
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

		if reply.Val != "" {
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
	return ""
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

func (this *Node) Republish(key string, value string) bool {
	kClosest := this.FindNode(DHT.GetHash(key))
	//use existing expire time
	expireTime := this.expireTime[key]

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

func (this *Node) Get(key string) (bool, string) {
	val := this.FindValue(DHT.GetHash(key), key)
	return val != "", val
}

func (this *Node) expireCheck() {
	for {
		this.check()
		time.Sleep(checkInterval)
	}
}

func (this *Node) check() {
	this.dataMux.RLock()
	for key := range this.data {
		if this.expireTime[key].Before(time.Now()) {
			this.dataMux.RUnlock()
			this.dataMux.Lock()

			delete(this.data, key)
			delete(this.expireTime, key)

			this.dataMux.Unlock()
			this.dataMux.RLock()
		}
	}
	this.dataMux.RUnlock()

}

func (this *Node) republishKey() {
	for {
		for key, value := range this.data {
			_, ok := this.republish[key]
			if ok && this.republish[key] == false {
				this.republish[key] = true
				continue
			}
			this.Republish(key, value)
		}
		time.Sleep(republishInterval)
	}
}
