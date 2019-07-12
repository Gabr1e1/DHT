package DHT

import (
	"fmt"
	"log"
	"math"
	"math/rand"
	"net"
	"net/rpc"
	"sync"
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
	Info        InfoType
	data        map[int]KVPair
	server      *rpc.Server
	mux         sync.Mutex
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
	n.Info = InfoType{addr, getHash(addr)}
	fmt.Println("INFO: ", n.Info)
	n.data = make(map[int]KVPair)
	for i := 0; i < M; i++ {
		n.Finger[i] = n.Info
	}
}

func (n *Node) Run() {
	n.server = rpc.NewServer()
	n.server.Register(n)

	listener, err := net.Listen("tcp", n.Info.IPAddr)
	if err != nil {
		log.Fatal("listen error: ", err)
	}
	go n.server.Accept(listener)
	go n.stablize()
	go n.fixFingers()
}

func (n *Node) findPredecessor(id int) InfoType {
	//fmt.Println(n.Info, "Finding Predecessor")
	p := n.Info
	successor := n.Finger[0]

	for !checkBetween(p.NodeNum+1, successor.NodeNum, id) {
		//fmt.Println("Going to check: ", p)
		var err error
		if p != n.Info {
			client := n.Connect(p)
			err = client.Call("Node.ClosestPrecedingFinger", &id, &p)
			client.Close()
		} else {
			err = n.ClosestPrecedingFinger(&id, &p)
			//fmt.Println(n.Info, p, id)
		}
		if err != nil {
			log.Fatal("Can't find Predecessor: ", err)
			return InfoType{}
		}
		client := n.Connect(p)
		client.Call("Node.GetSuccessor", 0, &successor)
		client.Close()

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
		if checkBetween(n.Info.NodeNum+1, *id, n.Finger[i].NodeNum) {
			*reply = n.Finger[i]
			//if i != 0 {
			//	fmt.Println("using finger: ", i, n.Info.NodeNum+1, *id, n.Finger[i].NodeNum)
			//}
			return nil
		}
	}
	*reply = n.Info
	return nil
}

func (n *Node) FindSuccessor(id *int, reply *InfoType) error {
	n.mux.Lock()
	t := n.findPredecessor(*id)
	client := n.Connect(t)
	err := client.Call("Node.GetSuccessor", 0, reply)
	n.mux.Unlock()
	client.Close()

	if err != nil {
		log.Fatal("Can't get successor: ", err)
		return err
	}
	return nil
}

func (n *Node) Get(k string) (bool, string) {
	var val string
	n.Get_(&k, &val)
	return val != "", val
}

func (n *Node) Get_(k *string, reply *string) error {
	id := getHash(*k)
	if val, ok := n.data[id]; ok {
		*reply = val.Value
	}
	var p InfoType
	n.FindSuccessor(&id, &p)
	if p != n.Info {
		client := n.Connect(p)
		var res string
		err := client.Call("Node.Get_", k, &res)
		if err != nil {
			client.Close()
			fmt.Println("Can't get Node: ", err)
			return err
		}
		client.Close()
		*reply = res
	}
	return nil
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
	if p == n.Info {
		n.data[id] = KVPair{kv.Key, kv.Value}
		*reply = true
	} else {
		client := n.Connect(p)
		err := client.Call("Node.Put_", kv, reply)
		if err != nil {
			client.Close()
			fmt.Println("Can't Put data in another node: ", err)
			return err
		}
		client.Close()
	}
	return nil
}

func (n *Node) Del(k string) bool {
	var flg bool
	n.Del_(&k, &flg)
	return flg
}

func (n *Node) Del_(k *string, reply *bool) error {
	id := getHash(*k)
	var p InfoType
	n.FindSuccessor(&id, &p)
	if p == n.Info {
		_, ok := n.data[id]
		if ok {
			delete(n.data, id)
		}
		*reply = ok
	} else {
		client := n.Connect(p)
		err := client.Call("Node.Del_", k, reply)
		if err != nil {
			client.Close()
			fmt.Println("Can't Delete data in another node: ", err)
			return err
		}
		client.Close()
	}
	return nil
}

func (n *Node) Ping(addr string) bool {
	client, err := rpc.Dial("tcp", addr)
	if err != nil {
		return false
	}
	var success int
	client.Call("Node.GetStatus", nil, &success)
	client.Close()
	return success > 0
}

func (n *Node) GetStatus(reply *int) {
	*reply = 1
}

func (n *Node) Dump() {
	fmt.Println("Num: ", n.Info.NodeNum)
	fmt.Println("Predecessor: ", n.Predecessor)
	fmt.Println("Successor: ", n.Finger[0])
}

func (n *Node) updateNode(other InfoType) InfoType {
	client := n.Connect(other)
	var newNode InfoType
	client.Call("Node.GetNodeInfo", nil, &newNode)
	client.Close()
	return newNode
}

func (n *Node) GetNodeInfo(_ *int, reply *InfoType) error {
	*reply = n.Info
	return nil
}

func (n *Node) DirectPut(KV *KVPair, reply *int) error {
	n.data[getHash(KV.Key)] = *KV
	return nil
}

//concurrency problem?????
func (n *Node) TransferData(replace *InfoType, reply *int) error {
	n.mux.Lock()
	client := n.Connect(*replace)
	for hashKey, KV := range n.data {
		if checkBetween(n.Predecessor.NodeNum, replace.NodeNum, hashKey) {
			var tmp int
			err := client.Call("Node.DirectPut", &KV, &tmp)
			if err != nil {
				n.mux.Unlock()
				client.Close()
				log.Fatal("Transfer Failed", err)
				return err
			}
		}
	}
	n.mux.Unlock()
	client.Close()
	return nil
}

//Join n itself to the network which addr belongs
func (n *Node) Join(addr string) bool {
	client, err := rpc.Dial("tcp", addr)
	if err != nil {
		fmt.Println("Can't Connect while attempting to join: ", err)
		return false
	}
	var other InfoType
	err = client.Call("Node.GetNodeInfo", 0, &other)
	if err != nil {
		fmt.Println("Can't Join: ", err)
		return false
	}

	n.mux.Lock()
	err = client.Call("Node.FindSuccessor", &n.Info.NodeNum, &n.Finger[0])
	n.mux.Unlock()
	client.Close()

	client = n.Connect(n.Finger[0])
	if client == nil {
		fmt.Println("Can't Connect to successor while joining: ", n.Finger[0])
		return false
	}
	var tmp int
	err = client.Call("Node.Notify", &n.Info, &tmp)
	if err != nil {
		fmt.Println("Can't notify other node: ", err)
		return false
	}
	err = client.Call("Node.TransferData", &n.Info, &tmp)
	if err != nil {
		fmt.Println("Can't transfer data: ", err)
		return false
	}
	client.Close()
	return true
}

//verify(and possibly change) n's successor
func (n *Node) stablize() {
	for {
		var x InfoType
		//fmt.Println("Stabilize call")
		client := n.Connect(n.Finger[0])
		if client == nil {
			continue
		}
		err := client.Call("Node.GetPredecessor", 0, &x)
		if err != nil {
			fmt.Println("Can't get predecessor: ", err)
			continue
		}
		//fmt.Println(n.Info.NodeNum, n.Finger[0].NodeNum, x.NodeNum)

		n.mux.Lock()
		if x.NodeNum != 0 && checkBetween(n.Info.NodeNum+1, n.Finger[0].NodeNum, x.NodeNum) {
			n.Finger[0] = x
			fmt.Printf("STABLIZE: %d's successor is %d\n", n.Info.NodeNum, x.NodeNum)
		}
		n.mux.Unlock()

		client.Close()

		client = n.Connect(n.Finger[0])
		if client == nil {
			continue
		}
		var tmp int
		err = client.Call("Node.Notify", &n.Info, &tmp)
		if err != nil {
			fmt.Println("Can't Notify: ", err)
			continue
		}
		client.Close()
		time.Sleep(1000 * time.Millisecond)
	}
}

//n(self) is notified of the existence of other which is a candidate for Predecessor
func (n *Node) Notify(other *InfoType, reply *int) error {
	n.mux.Lock()
	if n.Predecessor.IPAddr == "" || checkBetween(n.Predecessor.NodeNum+1, n.Info.NodeNum, other.NodeNum) {
		n.Predecessor = *other
		fmt.Printf("NOTIFY: %d's predecessor is %d\n", n.Info.NodeNum, other.NodeNum)
	}
	n.mux.Unlock()
	*reply = 0
	return nil
}

func (n *Node) fixFingers() {
	for {
		//fmt.Printf("Fixing Fingers for node %d\n", n.Info.NodeNum)
		i := rand.Intn(M-1) + 1 //random numbe // r in [1, M - 1]
		id := n.Info.NodeNum + int(math.Pow(2, float64(i)))
		id = id % (int(math.Pow(2, float64(M))))
		n.FindSuccessor(&id, &n.Finger[i])
		time.Sleep(1000 * time.Millisecond)
	}
}
