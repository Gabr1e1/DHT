package app

import (
	"../src"
	"debug/pe"
	"fmt"
	"log"
	"math"
	"math/big"
	"net/http/pprof"
	"os"
)

type Peer struct {
	node          *DHT.Node
	connectedPeer map[string]bool
	pieces        map[int]bool
	totalPieces   int
	file          *os.File
}

func decToHex(x *big.Int) string {
	return fmt.Sprintf("%x", x)
}

func (this *Peer) Publish(fileName string) {
	var err error
	this.file, err = os.Open(fileName)
	if err !=  nil {
		log.Fatal("Can't open file")
	}

	fmt.Println("magnet:?hash=", decToHex(getFileHash(this.file)))
	this.node = new(DHT.Node)
	this.node.Create(DHT.GetLocalAddress() + ":2000")

	t,  _ := this.file.Stat()
	this.totalPieces = int(t.Size() / pieceSize) +  (t.Size() )
	for i := 0; i <
}

func (this *Peer) Run() {
	if this.node == nil {
		log.Fatal("Haven't created node yet")
	}
	this.node.Run()
}

func (this *Peer) Download(link string) {

}
