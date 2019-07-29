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

type Contact struct {
	IPAddr  string
	NodeNum *big.Int
}

type KVPair struct {
	key   string
	value string
}

type Node struct {
	bucket     []KBucket
	self       Contact
	data       map[string]string
	expireTime map[string]time.Time //data automatically expires after expire time
	dataMux    sync.RWMutex

	server   *rpc.Server
	listener net.Listener
}

func (this *Node) check() {
	for {
		this.checkExpire()
		time.Sleep(time.Second)
	}
}

func (this *Node) checkExpire() {
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

func (this *Node) Create(addr string) {
	this.self = Contact{addr, DHT.GetHash(addr)}
}

func (this *Node) Run() {
	this.server = rpc.NewServer()
	_ = this.server.Register(this)

	var err error = nil
	this.listener, err = net.Listen("tcp", this.self.IPAddr)
	if err != nil {
		log.Fatal("listen error: ", err)
	}
}

func (this *Node) Ping(contact Contact) bool {
	client, err := this.Connect(contact)
	if err != nil {
		return false
	}
	var success bool
	err = client.Call("Node.RPCPing", &this.self, &success)
	if err != nil {
		return false
	}
	return success
}

func (this *Node) GetClosest(key *big.Int, num int) []Contact {
	l := this.CalcPrefix(key)
	cur := this.bucket[l].Get(alpha)
	if len(cur) < alpha {
		var t []Contact
		for i := 0; i < len(this.bucket); i++ {
			if i == l {
				continue
			}
			t = append(t, this.bucket[i].contacts...)
		}
		cur = append(cur, this.GetClosestInList(K-len(cur), t)...)
	}
	return cur
}

func (this *Node) FindNode(key *big.Int) []Contact {
	cur := this.GetClosest(key, K)
	var ans []Contact
	for len(cur) > 0 {
		contact := cur[0]
		cur = cur[1:]

		flg := false
		for _, i := range ans {
			if i.IPAddr == contact.IPAddr {
				flg = true
				break
			}
		}
		if flg {
			continue
		}

		ans = append(ans, contact)
		client, err := this.Connect(contact)
		if err != nil {
			fmt.Println(this.self, "Can't call", contact)
			continue
		}
		var reply FindNodeReturn
		err = client.Call("Node.FindNode", &FindNodeRequest{this.self, key}, &reply)
		if err != nil {
			fmt.Println(this.self, "Can't call FindNode in", contact)
			continue
		}
		cur = append(cur, reply.closest...)
	}
	return this.GetClosestInList(K, ans)
}

func (this *Node) FindValue(key *big.Int) string {
	cur := this.GetClosest(key, K)
	var ans []Contact
	for len(cur) > 0 {
		contact := cur[0]
		cur = cur[1:]

		flg := false
		for _, i := range ans {
			if i.IPAddr == contact.IPAddr {
				flg = true
				break
			}
		}
		if flg {
			continue
		}

		ans = append(ans, contact)
		client, err := this.Connect(contact)
		if err != nil {
			fmt.Println(this.self, "Can't call", contact)
			continue
		}
		var reply FindValueReturn
		err = client.Call("Node.FindNode", &FindNodeRequest{this.self, key}, &reply)
		if err != nil {
			fmt.Println(this.self, "Can't call FindNode in", contact)
			continue
		}
		if reply.val != "" {
			return reply.val
		}
		cur = append(cur, reply.closest...)
	}
	return ""
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
		err = client.Call("Node.RPCStore", &StoreRequest{this.self, KVPair{key, value}, expireTime}, &reply)
		if err != nil || !reply.success || verifyIdentity(kClosest[i], reply.self) {
			return false
		}
	}
	return true
}

func (this *Node) Get(key string) (bool, string) {
	val := this.FindValue(DHT.GetHash(key))
	return val != "", val
}
