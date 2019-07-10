package main

import (
	"fmt"
	"log"
	"math"
	"math/rand"
	"net"
	"net/http"
	"net/rpc"
)

const M = 32
const size = 1000

type KVPair struct {
	key   string
	value string
}

type Node struct {
	finger      [M]*Node
	nodeNum     int
	predecessor *Node
	IPAddr      string
	data        map[int]KVPair
}

//check whether c in [a,b)
func checkBetween(a, b, mid int) bool {
	if a > b {
		b += M
	}
	return mid >= a && mid < b
}

// create a dht-net with this node as start node
func (n *Node) Create() {
	for i := 0; i < M; i++ {
		n.finger[i] = n
	}
}

func (n *Node) Run() {
	node := new(Node)
	n.data = make(map[int]KVPair)

	rpc.Register(node)
	rpc.HandleHTTP()
	listener, err := net.Listen("tcp", n.IPAddr)
	if err != nil {
		log.Fatal("listen error: ", err)
	}
	go http.Serve(listener, nil)
	go n.stablize()
}

func (n *Node) findPredecessor(id int) *Node {
	p := n
	for checkBetween(p.nodeNum, p.finger[0].nodeNum, id) {
		client := n.connect(p)
		client.Call("Node.closestPrecedingFinger", id, p)
	}
	return p
}

func (n *Node) closestPrecedingFinger(id int, ret *Node) error {
	for i := M; i >= 1; i-- {
		if checkBetween(n.nodeNum, id, n.finger[i].nodeNum) {
			*ret = *n.finger[i]
		}
	}
	return nil
}

func (n *Node) findSuccessor(id int) *Node {
	t := n.findPredecessor(id)
	return t.finger[0]
}

func (n *Node) Get(k string) (bool, string) {
	var val string
	n._Get(&k, &val)
	return val == "", val
}

func (n *Node) _Get(k *string, reply *string) {
	id := getHash(*k)
	if val, ok := n.data[id]; ok {
		*reply = val.value
	}
	p := n.findSuccessor(id)
	if p != n {
		client := n.connect(p)
		var res string
		client.Call("Node._Get", k, &res) //could be wrong!!!!
		*reply = res
	}
}

func (n *Node) Put(k string, v string) bool {
	var flg bool
	n._Put(&KVPair{k, v}, &flg)
	return flg
}

func (n *Node) _Put(kv *KVPair, reply *bool) {
	id := getHash(kv.key)
	p := n.findSuccessor(id)
	if p == n {
		n.data[id] = KVPair{kv.key, kv.value}
		*reply = true
	} else {
		client := n.connect(p)
		client.Call("Node.Put", kv, reply)
	}
}

func (n *Node) Del(k string) bool {
	var flg bool
	n._Del(&k, &flg)
	return flg
}

func (n *Node) _Del(k *string, reply *bool) {
	id := getHash(*k)
	p := n.findSuccessor(id)
	if p == n {
		_, ok := n.data[id]
		if ok {
			delete(n.data, id)
		}
		*reply = ok
	} else {
		client := n.connect(p)
		client.Call("Node.Del", k, reply)
	}
}

func (n *Node) Ping(addr string) bool {
	client, err := rpc.DialHTTP("tcp", addr)
	if err != nil {
		return false
	}
	var success int
	client.Call("Node.GetStatus", nil, &success)
	return success > 0
}

func (n *Node) GetStatus(reply *int) {
	*reply = 1
}

func (n *Node) Dump() {
	fmt.Println("Num: ", n.nodeNum)
	fmt.Println("Predecessor: ", n.predecessor)
	fmt.Println("Successor: ", n.finger[0])
}

func (n *Node) updateNode(other *Node) *Node {
	client := n.connect(other)
	var newNode Node
	client.Call("Node._GetNode", nil, &newNode)
	return &newNode
}

func (n *Node) _GetNode(reply *Node) {
	*reply = *n
}

//Join n itself to the network which addr belongs
func (n *Node) Join(addr string) bool {
	client, err := rpc.DialHTTP("tcp", addr)
	if err != nil {
		return false
	}
	var other Node
	client.Call("Node._GetNode", nil, &other)

	n.predecessor = nil
	n.finger[0] = &other
	return true
}

//verify(and possibly change) n's successor
func (n *Node) stablize() {
	n.finger[0] = n.updateNode(n.finger[0]) //needs communication???
	x := n.finger[0].predecessor
	if checkBetween(n.nodeNum, n.finger[0].nodeNum, x.nodeNum) {
		n.finger[0] = x
	}
	client := n.connect(n.finger[0])
	client.Call("Node.notify", n, nil)
}

//n(self) is notified of the existence of other which is a candidate for predecessor
func (n *Node) notify(other *Node) {
	if n.predecessor == nil || checkBetween(n.predecessor.nodeNum, n.nodeNum, other.nodeNum) {
		*n.predecessor = *other // ???
	}
}

func (n *Node) fixFingers() {
	i := rand.Intn(M-1) + 1 //random number in [1, M - 1]
	n.finger[i] = n.findSuccessor(n.nodeNum + int(math.Pow(2, float64(i))))
}
