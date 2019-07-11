package DHT

import (
	"fmt"
	"log"
	"math"
	"math/rand"
	"net"
	"net/rpc"
	"time"
)

const M = 32
const size = 1000

type KVPair struct {
	Key   string
	Value string
}

type InfoType struct {
	IPAddr  string
	NodeNum int
}

type Node struct {
	Finger      [M]InfoType
	Predecessor InfoType
	info        InfoType
	data        map[int]KVPair
	server      *rpc.Server
}

//check whether c in [a,b)
func checkBetween(a, b, mid int) bool {
	if a >= b {
		return checkBetween(a, int(math.Pow(2, M)), mid) || checkBetween(0, b, mid)
	}
	return mid >= a && mid < b
}

// create a dht-net with this node as start node
func (n *Node) Create(addr string) {
	n.info = InfoType{addr, getHash(addr)}
	n.data = make(map[int]KVPair)
	for i := 0; i < M; i++ {
		n.Finger[i] = n.info
	}
}

func (n *Node) Run() {
	n.server = rpc.NewServer()
	n.server.Register(n)

	listener, err := net.Listen("tcp", n.info.IPAddr)
	if err != nil {
		log.Fatal("listen error: ", err)
	}
	go n.server.Accept(listener)
	go n.stablize()
}

func (n *Node) findPredecessor(id int) InfoType {
	//fmt.Println("Finding Predecessor")
	p := n.info
	successor := n.Finger[0]

	for !checkBetween(p.NodeNum+1, successor.NodeNum-1, id) {
		var err error
		if p != n.info {
			client := n.connect(p)
			err = client.Call("Node.ClosestPrecedingFinger", &id, &p)
		} else {
			err = n.ClosestPrecedingFinger(&id, &p)
			//fmt.Println(n.info, p, id)
		}
		if err != nil {
			log.Fatal("Can't find Predecessor: ", err)
			return InfoType{}
		}
		client := n.connect(p)
		client.Call("Node.GetSuccessor", 0, &successor)
	}
	//fmt.Println("Found Predecessor", p)
	return p
}

func (n *Node) GetSuccessor(tmp *int, reply *InfoType) error {
	*reply = n.Finger[0]
	return nil
}

func (n *Node) GetPredecessor(tmp *int, reply *InfoType) error {
	*reply = n.Predecessor
	return nil
}

func (n *Node) ClosestPrecedingFinger(id *int, reply *InfoType) error {
	for i := M - 1; i >= 0; i-- {
		if checkBetween(n.info.NodeNum+1, *id, n.Finger[i].NodeNum) {
			*reply = n.Finger[i]
			return nil
		}
	}
	return nil
}

func (n *Node) FindSuccessor(id *int, reply *InfoType) error {
	t := n.findPredecessor(*id)
	client := n.connect(t)
	err := client.Call("Node.GetSuccessor", 0, reply)
	if err != nil {
		log.Fatal("Can't get successor: ", err)
		return err
	}
	return nil
}

func (n *Node) Get(k string) (bool, string) {
	var val string
	n._Get(&k, &val)
	return val != "", val
}

func (n *Node) _Get(k *string, reply *string) {
	id := getHash(*k)
	if val, ok := n.data[id]; ok {
		*reply = val.Value
	}
	var p InfoType
	n.FindSuccessor(&id, &p)
	if p != n.info {
		client := n.connect(p)
		var res string
		client.Call("Node._Get", k, &res) //could be wrong!!!!
		*reply = res
	}
}

func (n *Node) Put(k string, v string) bool {
	var flg bool
	n.Put_(&KVPair{k, v}, &flg)
	return flg
}

func (n *Node) Put_(kv *KVPair, reply *bool) error {
	id := getHash(kv.Key)
	var p InfoType
	n.FindSuccessor(&id, &p)
	if p == n.info {
		n.data[id] = KVPair{kv.Key, kv.Value}
		*reply = true
	} else {
		client := n.connect(p)
		err := client.Call("Node.Put_", kv, reply)
		if err != nil {
			fmt.Println("Can't Put data in another node: ", err)
			return err
		}
	}
	return nil
}

func (n *Node) Del(k string) bool {
	var flg bool
	n._Del(&k, &flg)
	return flg
}

func (n *Node) _Del(k *string, reply *bool) {
	id := getHash(*k)
	var p InfoType
	n.FindSuccessor(&id, &p)
	if p == n.info {
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
	client, err := rpc.Dial("tcp", addr)
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
	fmt.Println("Num: ", n.info.NodeNum)
	fmt.Println("Predecessor: ", n.Predecessor)
	fmt.Println("Successor: ", n.Finger[0])
}

func (n *Node) updateNode(other InfoType) InfoType {
	client := n.connect(other)
	var newNode InfoType
	client.Call("Node.GetNodeInfo", nil, &newNode)
	return newNode
}

func (n *Node) GetNodeInfo(_ *int, reply *InfoType) error {
	*reply = n.info
	return nil
}

func (n *Node) DirectPut(KV *KVPair, reply *int) error {
	n.data[getHash(KV.Key)] = *KV
	return nil
}

//concurrency problem?????
func (n *Node) TransferData(replace *InfoType, reply *int) error {
	client := n.connect(*replace)
	for hashKey, KV := range n.data {
		if checkBetween(n.Predecessor.NodeNum, replace.NodeNum, hashKey) {
			var tmp int
			err := client.Call("Node.DirectPut", &KV, &tmp)
			if err != nil {
				log.Fatal("Transfer Failed", err)
				return err
			}
		}
	}
	return nil
}

//Join n itself to the network which addr belongs
func (n *Node) Join(addr string) bool {
	client, err := rpc.Dial("tcp", addr)
	if err != nil {
		return false
	}
	var other InfoType
	err = client.Call("Node.GetNodeInfo", 0, &other)
	if err != nil {
		log.Fatal("Can't Join: ", err)
		return false
	}
	n.Predecessor = InfoType{"", 0}
	err = client.Call("Node.FindSuccessor", n.info.NodeNum, &n.Finger[0])

	client = n.connect(n.Finger[0])
	var tmp int
	err = client.Call("Node.Notify", &n.info, &tmp)
	if err != nil {
		log.Fatal("Can't notify other node: ", err)
		return false
	}
	err = client.Call("Node.TransferData", &n.info, &tmp)
	if err != nil {
		log.Fatal("Can't transfer data: ", err)
		return false
	}
	return true
}

//verify(and possibly change) n's successor
func (n *Node) stablize() {
	for {
		var x InfoType
		client := n.connect(n.Finger[0])
		client.Call("Node.GetPredecessor", 0, &x)
		//fmt.Println(n.info.NodeNum, n.Finger[0].NodeNum, x.NodeNum)

		if x.NodeNum != 0 && checkBetween(n.info.NodeNum+1, n.Finger[0].NodeNum, x.NodeNum) {
			n.Finger[0] = x
			fmt.Printf("STABLIZE: %d's successor is %d\n", n.info.NodeNum, x.NodeNum)
		}
		client = n.connect(n.Finger[0])
		var tmp int
		err := client.Call("Node.Notify", &n.info, &tmp)
		if err != nil {
			fmt.Println("Can't Notify: ", err)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

//n(self) is notified of the existence of other which is a candidate for Predecessor
func (n *Node) Notify(other *InfoType, reply *int) error {
	if n.Predecessor.IPAddr == "" || checkBetween(n.Predecessor.NodeNum+1, n.info.NodeNum, other.NodeNum) {
		n.Predecessor = *other
		fmt.Printf("NOTIFY: %d's predecessor is %d\n", n.info.NodeNum, other.NodeNum)
	}
	*reply = 0
	return nil
}

func (n *Node) fixFingers() {
	i := rand.Intn(M-1) + 1 //random number in [1, M - 1]
	id := n.info.NodeNum + int(math.Pow(2, float64(i)))
	n.FindSuccessor(&id, &n.Finger[i])
}
