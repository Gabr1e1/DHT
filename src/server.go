package DHT

import (
	"fmt"
	"log"
	"math/big"
	"net"
	"net/rpc"
	"strconv"
)

const duplicateNum = 1

// create a dht-net with this node as start node
func (n *Node) Create(addr string) {
	var t = GetHash(addr)
	n.Info = InfoType{addr, t}
	n.Predecessor = InfoType{"", big.NewInt(0)}
	fmt.Println("INFO: ", n.Info)
	if len(n.data) == 0 {
		n.data = make(map[string]KVPair)
	}
	for i := 0; i < M; i++ {
		n.Finger[i], n.Successors[i] = copyInfo(n.Info), copyInfo(n.Info)
	}
}

func (n *Node) Run() {
	n.status = 1
	n.server = rpc.NewServer()
	_ = n.server.Register(n)

	var err error = nil
	n.listener, err = net.Listen("tcp", n.Info.IPAddr)
	if err != nil {
		log.Fatal("listen error: ", err)
	}
	go n.server.Accept(n.listener)
	go n.stabilize()
	go n.fixFingers()
}

func (n *Node) Get(k string) (bool, string) {
	var val string
	for i := 0; i < duplicateNum; i++ {
		var newk = k + "?" + strconv.Itoa(i)
		err := n.Get_(&newk, &val)
		if err == nil {
			return val != "", val
		}
	}
	return val != "", val
}

func (n *Node) Put(k string, v string) bool {
	var flg bool
	for i := 0; i < duplicateNum; i++ {
		var newk = k + "?" + strconv.Itoa(i)
		_ = n.Put_(&KVPair{newk, v}, &flg)
		if !flg {
			return false
		}
	}
	return true
}

func (n *Node) Del(k string) bool {
	var flg bool
	for i := 0; i < duplicateNum; i++ {
		var newk = k + "?" + strconv.Itoa(i)
		_ = n.Del_(&newk, &flg)
	}
	return true
}

//TODO: Append Directly
func (n *Node) AppendTo(k string, v string) {
	ok, t := n.Get(k)
	if ok {
		n.Put(k, t+v)
	}
}

func (n *Node) Ping(addr string) bool {
	//otherwise could be a dead lock
	if addr == "" {
		return false
	}
	if addr == n.Info.IPAddr {
		return n.status > 0
	}

	client, err := n.Connect(InfoType{addr, big.NewInt(0)})
	if err != nil {
		fmt.Println("Ping Failed", addr)
		return false
	}

	var success int
	err = client.Call("Node.GetStatus", 0, &success)
	if err != nil {
		fmt.Println("GetStatus Error: ", err)
		_ = client.Close()
		return false
	}
	_ = client.Close()
	return success > 0
}

func (n *Node) GetStatus(_ *int,
	reply *int) error {
	*reply = n.status
	return nil
}

func (n *Node) Dump() {
	fmt.Println("Address: ", n.Info.IPAddr)
	fmt.Println("Num: ", n.Info.NodeNum)
	fmt.Println("Predecessor: ", n.Predecessor)
	fmt.Println("Successor: ", n.Successors)
	fmt.Println("Data: ")
	for _, v := range n.data {
		fmt.Print(v)
	}
	fmt.Println()
}

//Join n itself to the network which addr belongs
func (n *Node) Join(addr string) bool {
	client, err := n.Connect(InfoType{addr, big.NewInt(0)})
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
	err = client.Call("Node.FindSuccessor", n.Info.NodeNum, &n.Successors[0])
	n.Finger[0] = copyInfo(n.Successors[0])
	n.mux.Unlock()
	_ = client.Close()

	client, err = n.Connect(n.Successors[0])
	if err != nil {
		fmt.Println("Can't Connect to successor while joining: ", n.Successors[0])
		return false
	}
	var tmp int
	err = client.Call("Node.Notify", &n.Info, &tmp)
	if err != nil {
		fmt.Println("Can't notify other node: ", err)
		return false
	}
	var pred InfoType
	err = client.Call("Node.GetPredecessor", 0, &pred)
	err = client.Call("Node.TransferData", &pred, &tmp)
	if err != nil {
		fmt.Println("Can't transfer data: ", err)
		return false
	}
	_ = client.Close()
	return true
}

func (n *Node) Quit() {
	var tmp int

	n.mux.Lock()
	err := n.FindFirstSuccessorAlive(nil, &n.Successors[0])
	n.mux.Unlock()
	if err != nil {
		log.Fatal("Quit failed")
	}

	err = n.TransferDataForce(&n.Successors[0], &tmp)
	if err != nil {
		fmt.Println("Quit error: ", err)
		return
	}
	client, err := n.Connect(n.Predecessor)
	if err != nil {
		return
	}
	err = client.Call("Node.ModifySuccessors", &n.Successors[0], &tmp)
	_ = client.Close()
	if err != nil {
		fmt.Println("Quit error: ", err)
		return
	}

	client, err = n.Connect(n.Successors[0])
	if err != nil {
		return
	}
	err = client.Call("Node.ModifyPredecessor", &n.Predecessor, &tmp)
	_ = client.Close()
	if err != nil {
		fmt.Println("Quit error: ", err)
		return
	}

	_ = n.listener.Close()
	n.status = 0
}

func (n *Node) ForceQuit() {
	_ = n.listener.Close()
	n.status = 0
}
