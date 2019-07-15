package DHT

import (
	"fmt"
	"log"
	"net"
	"net/rpc"
)

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
	n.status = 1
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
	client, err := rpc.Dial("tcp", addr)
	if err != nil {
		return false
	}
	var success int
	client.Call("Node.GetStatus", nil, &success)
	client.Close()
	return success > 0
}

//1: F
func (n *Node) GetStatus(reply *int) {
	*reply = n.status
}

func (n *Node) Dump() {
	fmt.Println("Num: ", n.Info.NodeNum)
	fmt.Println("Predecessor: ", n.Predecessor)
	fmt.Println("Successor: ", n.Finger[0])
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

func (n *Node) Quit() {
	var tmp int
	err := n.TransferDataForce(&n.Finger[0], &tmp)
	if err != nil {
		fmt.Println("Quit error: ", err)
		return
	}
	client := n.Connect(n.Predecessor)
	err = client.Call("Node.ModifySuccessor", &n.Finger[0], &tmp)
	client.Close()
	if err != nil {
		fmt.Println("Quit error: ", err)
		return
	}

	client = n.Connect(n.Finger[0])
	err = client.Call("Node.ModifyPredecessor", &n.Predecessor, &tmp)
	client.Close()
	if err != nil {
		fmt.Println("Quit error: ", err)
		return
	}
}
