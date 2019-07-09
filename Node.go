package main

const M = 32
const size = 1000

type dataType struct {
	key   int
	value string
}

type Node struct {
	finger      [M]int
	nodeNum     int
	predecessor *Node
	successor   *Node
	data        [size]dataType //?
}

//check whether c in [a,b)
func checkBetween(a, b, c int) bool {
	return c >= a && c < b
}

// create a dht-net with this node as start node
func (n *Node) Create() {
	for i := 0; i < M; i++ {
		n.finger[i] = n.nodeNum
	}
}

func (n *Node) findPredecessor(id int) *Node {
	var p *Node = n
	for checkBetween(p.nodeNum, p.successor.nodeNum, id) {

	}
}

func (n *Node) findSuccessor(id int) *Node {
	t := n.findPredecessor(id)
	return t.successor
}

func (n *Node) Get(k string) (bool, string) {
	id := getHash(k)
	n.findSuccessor(id)
}
