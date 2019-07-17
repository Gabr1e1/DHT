package DHT

import (
	"fmt"
	"log"
	"net"
	"net/rpc"
)

// create a dht-net with this node as start node
func (n *Node) Create(addr string) {
	n.Info = InfoType{addr, GetHash(addr)}
	fmt.Println("INFO: ", n.Info)
	if len(n.data) == 0 {
		n.data = make(map[int]KVPair)
	}
	for i := 0; i < M; i++ {
		n.Finger[i], n.Successors[i] = n.Info, n.Info
	}
}

func (n *Node) Run() {
	n.status = 1
	n.server = rpc.NewServer()
	n.server.Register(n)

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
	n.Get_(&k, &val)
	return val != "", val
}

func (n *Node) Put(k string, v string) bool {
	var flg bool
	n.Put_(&KVPair{k, v}, &flg)
	return flg
}

func (n *Node) Del(k string) bool {
	var flg bool
	n.Del_(&k, &flg)
	return flg
}

func (n *Node) Ping(addr string) bool {
	//otherwise could be a dead lock
	if addr == "" {
		return false
	}
	if addr == n.Info.IPAddr {
		return n.status > 0
	}

	client, err := n.Connect(InfoType{addr, 0})
	if err != nil {
		//fmt.Println("Ping Failed", addr)
		return false
	}

	var success int
	err = client.Call("Node.GetStatus", 0, &success)
	if err != nil {
		fmt.Println("GetStatus Error: ", err)
		client.Close()
		return false
	}
	client.Close()
	return success > 0
}

//1: F
func (n *Node) GetStatus(_ *int, reply *int) error {
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
	client, err := n.Connect(InfoType{addr, 0})
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
	err = client.Call("Node.FindSuccessor", &n.Info.NodeNum, &n.Successors[0])
	n.Finger[0] = n.Successors[0]
	n.mux.Unlock()
	client.Close()

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
	client.Close()
	return true
}

func (n *Node) Quit() {
	var tmp int

	n.mux.Lock()
	n.FindFirstSuccessorAlive(nil, &n.Successors[0])
	n.mux.Unlock()
	err := n.TransferDataForce(&n.Successors[0], &tmp)
	if err != nil {
		fmt.Println("Quit error: ", err)
		return
	}
	client, err := n.Connect(n.Predecessor)
	if err != nil {
		return
	}
	err = client.Call("Node.ModifySuccessors", &n.Successors[0], &tmp)
	client.Close()
	if err != nil {
		fmt.Println("Quit error: ", err)
		return
	}

	client, err = n.Connect(n.Successors[0])
	if err != nil {
		return
	}
	err = client.Call("Node.ModifyPredecessor", &n.Predecessor, &tmp)
	client.Close()
	if err != nil {
		fmt.Println("Quit error: ", err)
		return
	}

	n.listener.Close()
	n.status = 0
}

func (n *Node) ForceQuit() {
	n.listener.Close()
	n.status = 0
}
