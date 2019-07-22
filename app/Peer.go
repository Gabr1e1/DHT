package app

import (
	"../src"
	"fmt"
	"log"
	"math"
	"math/big"
	"net"
	"net/rpc"
	"os"
	"strconv"
)

const maxPeerNum = 100

type PeerInfo struct {
	addr string
}

type Peer struct {
	Node           *DHT.Node
	availablePeers [maxPeerNum]PeerInfo
	pieces         map[int]bool
	totalPieces    int
	file           *os.File
	server         *rpc.Server
	Self           PeerInfo
}

func decToHex(x *big.Int) string {
	return fmt.Sprintf("%x", x)
}

func (this *Peer) Publish(fileName string, port1 int, port2 int, other string) string {
	var err error
	this.file, err = os.Open(fileName)
	if err != nil {
		log.Fatal("Can't open file: ", err)
	}

	var hash = decToHex(GetFileHash(this.file))
	var link = fmt.Sprint("magnet:?hash=" + hash + "?fn=" + fileName)
	fmt.Println(link)
	this.Self = PeerInfo{DHT.GetLocalAddress() + ":" + strconv.Itoa(port2)}

	this.Node = new(DHT.Node)
	if other != "" {
		this.Node.Join(other)
	}
	this.Node.Create(DHT.GetLocalAddress() + ":" + strconv.Itoa(port1))

	t, err := this.file.Stat()
	if err != nil {
		log.Fatal("Can't open file: ", err)
	}

	this.totalPieces = int(math.Ceil(float64(t.Size()) / pieceSize))
	if this.pieces == nil {
		this.pieces = make(map[int]bool)
	}
	for i := 0; i < this.totalPieces; i++ {
		this.pieces[i] = true
	}

	//no need to call Run() anymore
	this.Node.Run()
	this.Node.Put(hash, strconv.Itoa(this.totalPieces)+"?"+encodePeer(this.Self.addr))
	this.startServer()
	return link
}

func (this *Peer) startServer() {
	this.server = rpc.NewServer()
	_ = rpc.Register(this)
	_ = this.server.Register(this)

	listener, err := net.Listen("tcp", this.Self.addr)
	if err != nil {
		log.Fatal("listen error: ", err)
	}
	go this.server.Accept(listener)
}

func (this *Peer) Run(port1 int, port2 int, other string) {
	if this.Node == nil {
		this.Node = new(DHT.Node)
		this.Node.Create(DHT.GetLocalAddress() + ":" + strconv.Itoa(port1))
	}
	if this.pieces == nil {
		this.pieces = make(map[int]bool)
	}
	this.Self = PeerInfo{DHT.GetLocalAddress() + ":" + strconv.Itoa(port2)}
	this.startServer()
	this.Node.Run()
	if other != "" {
		this.Node.Join(other)
	}
}

func (this *Peer) nextToDownload() int {
	for i := 0; i < this.totalPieces; i++ {
		if this.pieces[i] == false {
			/*
				TODO: choose rarest peer
			*/
			return i
		}
	}
	return 0
}

func (this *Peer) CheckPiece(pieceNum *int, reply *bool) error {
	*reply = this.pieces[*pieceNum]
	return nil
}

func (this *Peer) getBestPeer(curPeer []PeerInfo) PeerInfo {
	/*
		TODO: use tit-tat strategy
	*/
	return curPeer[0]
}

func (this *Peer) download(pieceNum int) {
	/* Decide from which peer to download */
	var curPeer []PeerInfo
	for i := 0; i < maxPeerNum; i++ {
		if this.availablePeers[i].addr == "" {
			continue
		}
		client, err := this.Connect(this.availablePeers[i])
		if err != nil {
			log.Fatal("Download Failed: ", err)
		}
		var reply bool
		err = client.Call("Peer.CheckPiece", &pieceNum, &reply)
		client.Close()
		if err != nil {
			log.Fatal("Download Failed: ", err)
		}
		if reply {
			curPeer = append(curPeer, this.availablePeers[i])
		}
	}

	p := this.getBestPeer(curPeer)
	client, err := this.Connect(p)
	if err != nil {
		log.Fatal("Download Failed: ", err)
	}
	req := Request{this.Self.addr, pieceNum}
	var data []byte
	err = client.Call("Peer.UploadData", &req, &data)
	client.Close()
	if err != nil {
		log.Fatal("Download Failed: ", err)
	}
	err = writeToFile(this.file, data)
	if err != nil {
		log.Fatal("Can't write to file: ", err)
	}
	this.pieces[pieceNum] = true
}

func (this *Peer) Download(link string) bool {
	ok, hash, fileName := parseMagnetLink(link)
	fileName = fileName + ".download"
	if !ok {
		log.Fatal("Not a valid link")
	}
	success, value := this.Node.Get(hash)
	if !success {
		log.Fatal("Not a valid link")
	}

	this.file, _ = os.Create(fileName)
	if this.file == nil {
		log.Fatal("Can't create file")
	}
	/* Start download */
	/*
		TODO: Append self to the list
	*/

	var peer []PeerInfo
	peer, this.totalPieces = parsePeer(value)
	copy(this.availablePeers[:], peer)

	for len(this.pieces) != this.totalPieces {
		next := this.nextToDownload()
		this.download(next)
	}
	this.stripFile()

	/* Verify content */
	curHash := GetFileHash(this.file)
	if curHash.String() != hash {
		log.Fatal("Download Failed")
		return false
	}
	fmt.Println("Download successful")
	return true
}
