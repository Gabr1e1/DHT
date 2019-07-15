package DHT

import (
	"fmt"
	"log"
	"math"
	"math/rand"
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
	status      int
}

//check whether c in [a,b)
func checkBetween(a, b, mid int) bool {
	if a >= b {
		return checkBetween(a, int(math.Pow(2, M)), mid) || checkBetween(0, b, mid)
	}
	return mid >= a && mid < b
}

func (n *Node) findPredecessor(id int) InfoType {
	//cnt := 0
	p := n.Info
	successor := n.Finger[0]

	for !checkBetween(p.NodeNum+1, successor.NodeNum, id) {
		//fmt.Println("Going to check: ", p)
		//cnt++
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
	//fmt.Printf("Found Predecessor using %d jumps\n", cnt)
	return p
}

func (n *Node) GetSuccessor(tmp *int, reply *InfoType) error {
	*reply = n.Finger[0]
	return nil
}

func (n *Node) ModifySuccessor(succ *InfoType, reply *int) error {
	n.mux.Lock()
	n.Finger[0] = *succ
	n.mux.Unlock()
	return nil
}

func (n *Node) ModifyPredecessor(pred *InfoType, reply *int) error {
	n.mux.Lock()
	n.Predecessor = *pred
	n.mux.Unlock()
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

func (n *Node) TransferDataForce(replace *InfoType, reply *int) error {
	n.mux.Lock()
	client := n.Connect(*replace)
	for _, KV := range n.data {
		var tmp int
		err := client.Call("Node.DirectPut", &KV, &tmp)
		if err != nil {
			n.mux.Unlock()
			client.Close()
			log.Fatal("Transfer Failed", err)
			return err
		}
	}
	n.mux.Unlock()
	client.Close()
	return nil
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
		client.Close()
		if err != nil {
			fmt.Println("Can't Notify: ", err)
			continue
		}
		time.Sleep(1000 * time.Millisecond)
	}
}

//n(self) is notified of the existence of another node which is a candidate for Predecessor
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
