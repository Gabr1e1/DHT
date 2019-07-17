//TODO: MAXIMUM ROUTING NUMBER

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
const RoutingLimit = 1000

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
	Successors  [M]InfoType
	Predecessor InfoType
	Info        InfoType
	data        map[int]KVPair
	server      *rpc.Server
	mux         sync.Mutex
	status      int
	listener    net.Listener
	wg          sync.WaitGroup
}

//check whether c in [a,b)
func checkBetween(a, b, mid int) bool {
	if a >= b {
		return checkBetween(a, int(math.Pow(2, M)), mid) || checkBetween(0, b, mid)
	}
	return mid >= a && mid < b
}

//the successor list might not be effective due to force quitting nodes
func (n *Node) FindFirstSuccessorAlive(tmp *int, reply *InfoType) error {
	for i, node := range n.Successors {
		if !n.Ping(node.IPAddr) {
			n.Successors[i] = InfoType{}
			continue
		}
		*reply = node
		return nil
	}
	return nil //No successor
}

func (n *Node) GetSuccessors(_ *int, reply *[M]InfoType) error {
	*reply = n.Successors
	return nil
}

func (n *Node) ModifySuccessors(succ *InfoType, _ *int) error {
	if *succ == n.Info {
		return nil
	}
	client, err := n.Connect(*succ)
	if err != nil {
		return err
	}
	n.mux.Lock()
	var newSucList [M]InfoType
	err = client.Call("Node.GetSuccessors", 0, &newSucList)
	if err != nil {
		return err
	}
	_ = client.Close()
	n.Finger[0], n.Successors[0] = *succ, *succ
	for i := 1; i < M; i++ {
		n.Successors[i] = newSucList[i-1]
	}
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
	if n.Predecessor.NodeNum != 0 && !n.Ping(n.Predecessor.IPAddr) {
		n.Predecessor = InfoType{}
	}
	*reply = n.Predecessor
	return nil
}

func (n *Node) findPredecessor(id int) InfoType {
	//fmt.Println("Finding Predecessor")
	var cnt = 0

	p := n.Info
	var successor InfoType
	var tmp int
	err := n.FindFirstSuccessorAlive(&tmp, &successor)
	if err != nil {
		log.Fatal("Can't find predecessor")
		return InfoType{}
	}

	for !checkBetween(p.NodeNum+1, successor.NodeNum, id) && cnt <= RoutingLimit {
		cnt++
		var err error
		if p != n.Info {
			client, _ := n.Connect(p)
			err = client.Call("Node.ClosestPrecedingNode", &id, &p)
			if err != nil {
				return InfoType{}
			}

			_ = client.Close()
		} else {
			err = n.ClosestPrecedingNode(&id, &p)
			//fmt.Println(n.Info, p, id)
		}
		if err != nil {
			fmt.Println("Can't find Predecessor: ", err)
			return InfoType{}
		}
		client, _ := n.Connect(p)
		_ = client.Call("Node.FindFirstSuccessorAlive", 0, &successor)
		_ = client.Close()

	}
	//fmt.Printf("Found Predecessor using %d jumps\n", cnt)
	//fmt.Println("Found predecessor", p)
	return p
}

func (n *Node) ClosestPrecedingNode(id *int, reply *InfoType) error {
	//first check finger table
	for i := M - 1; i >= 0; i-- {
		if checkBetween(n.Info.NodeNum+1, *id, n.Finger[i].NodeNum) {
			// possible fail node
			if !n.Ping(n.Finger[i].IPAddr) {
				n.Finger[i] = InfoType{}
				continue
			}
			*reply = n.Finger[i]
			return nil
		}
	}

	//then check successor list
	for i := M - 1; i >= 0; i-- {
		if checkBetween(n.Info.NodeNum+1, *id, n.Successors[i].NodeNum) {
			// possible fail node
			if !n.Ping(n.Successors[i].IPAddr) {
				n.Successors[i] = InfoType{}
				continue
			}
			*reply = n.Successors[i]
			return nil
		}
	}

	*reply = n.Info
	return nil
}

func (n *Node) FindSuccessor(id *int, reply *InfoType) error {
	t := n.findPredecessor(*id)
	client, err := n.Connect(t)
	if err != nil {
		return err
	}
	n.mux.Lock()
	err = client.Call("Node.FindFirstSuccessorAlive", 0, reply)
	n.mux.Unlock()
	_ = client.Close()

	if err != nil {
		fmt.Println("Can't get successor: ", err)
		return err
	}
	return nil
}

func (n *Node) Get_(k *string, reply *string) error {
	id := GetHash(*k)
	if val, ok := n.data[id]; ok {
		*reply = val.Value
		return nil
	}
	var p InfoType
	err := n.FindSuccessor(&id, &p)
	if err != nil {
		return err
	}

	if p != n.Info {
		client, err := n.Connect(p)
		if err != nil {
			return err
		}
		var res string
		err = client.Call("Node.Get_", k, &res)
		if err != nil {
			_ = client.Close()
			fmt.Println("Can't get Node: ", err)
			return err
		}
		_ = client.Close()
		*reply = res
	}
	return nil
}

func (n *Node) Put_(kv *KVPair, reply *bool) error {
	id := GetHash(kv.Key)
	var p InfoType
	err := n.FindSuccessor(&id, &p)
	if err != nil {
		return err
	}

	if p == n.Info {
		n.data[id] = KVPair{kv.Key, kv.Value}
		*reply = true
	} else {
		client, err := n.Connect(p)
		if err != nil {
			return err
		}
		err = client.Call("Node.Put_", kv, reply)
		if err != nil {
			_ = client.Close()
			fmt.Println("Can't Put data in another node: ", err)
			return err
		}
		_ = client.Close()
	}
	return nil
}

func (n *Node) Del_(k *string, reply *bool) error {
	id := GetHash(*k)
	var p InfoType
	err := n.FindSuccessor(&id, &p)
	if err != nil {
		return err
	}

	if p == n.Info {
		_, ok := n.data[id]
		if ok {
			delete(n.data, id)
		}
		*reply = ok
	} else {
		client, err := n.Connect(p)
		if err != nil {
			return err
		}
		err = client.Call("Node.Del_", k, reply)
		if err != nil {
			_ = client.Close()
			fmt.Println("Can't Delete data in another node: ", err)
			return err
		}
		_ = client.Close()
	}
	return nil
}

func (n *Node) GetNodeInfo(_ *int, reply *InfoType) error {
	*reply = n.Info
	return nil
}

func (n *Node) DirectPut(KV *KVPair, reply *int) error {
	n.data[GetHash(KV.Key)] = *KV
	return nil
}

func (n *Node) TransferData(replace *InfoType, reply *int) error {
	if replace.IPAddr == "" {
		return nil
	}

	client, err := n.Connect(*replace)
	if err != nil {
		return err
	}

	n.mux.Lock()
	for hashKey, KV := range n.data {
		if checkBetween(n.Info.NodeNum, replace.NodeNum, hashKey) {
			var tmp int
			err := client.Call("Node.DirectPut", &KV, &tmp)
			if err != nil {
				n.mux.Unlock()
				_ = client.Close()
				fmt.Println("Transfer Failed", err)
				return err
			}
			delete(n.data, hashKey)
		}
	}
	n.mux.Unlock()
	_ = client.Close()
	return nil
}

func (n *Node) TransferDataForce(replace *InfoType, reply *int) error {
	client, err := n.Connect(*replace)
	if err != nil {
		return err
	}

	n.mux.Lock()
	for _, KV := range n.data {
		var tmp int
		err := client.Call("Node.DirectPut", &KV, &tmp)
		if err != nil {
			n.mux.Unlock()
			_ = client.Close()
			fmt.Println("Transfer Failed", err)
			return err
		}
	}
	n.mux.Unlock()
	_ = client.Close()
	return nil
}

//verify(and possibly change) n's successor
func (n *Node) stabilize() {
	for {
		if n.status == 0 {
			break
		}
		var tmp int
		var x InfoType

		n.mux.Lock()
		err := n.FindFirstSuccessorAlive(nil, &n.Successors[0])
		n.mux.Unlock()
		if err != nil {
			continue
		}
		//fmt.Println("First Successor alive of ", n.Info, " is ", n.Successors[0])

		client, err := n.Connect(n.Successors[0])
		if err != nil {
			continue
		}
		err = client.Call("Node.GetPredecessor", 0, &x)
		if err != nil {
			fmt.Println("Can't get predecessor: ", err)
			continue
		}
		//fmt.Println(n.Info.NodeNum, n.Successors[0].NodeNum, x.NodeNum)

		n.mux.Lock()
		if x.NodeNum != 0 && checkBetween(n.Info.NodeNum+1, n.Successors[0].NodeNum, x.NodeNum) {
			n.Successors[0], n.Finger[0] = x, x
			fmt.Printf("STABILIZE: %d's successor is %d\n", n.Info.NodeNum, x.NodeNum)
		}
		n.mux.Unlock()
		_ = client.Close()

		err = n.ModifySuccessors(&n.Successors[0], &tmp)
		if err != nil {
			continue
		}
		client, err = n.Connect(n.Successors[0])
		if err != nil {
			continue
		}

		err = client.Call("Node.Notify", &n.Info, &tmp)
		_ = client.Close()
		if err != nil {
			fmt.Println("Can't Notify: ", err)
			continue
		}
		time.Sleep(333 * time.Millisecond)
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
		if n.status == 0 {
			break
		}
		i := rand.Intn(M-1) + 1 //random numbers in [1, M - 1]
		id := n.Info.NodeNum + int(math.Pow(2, float64(i)))
		id = id % (int(math.Pow(2, float64(M))))
		err := n.FindSuccessor(&id, &n.Finger[i])
		if err != nil {
			continue
		}
		time.Sleep(666 * time.Millisecond)
	}
}
