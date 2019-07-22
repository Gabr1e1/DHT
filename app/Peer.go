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
const maxPieceNum = 65536

type PeerInfo struct {
	addr string
}

type PieceInfo struct {
	have   bool
	hash   *big.Int
	length int
	isLast bool
}

type Peer struct {
	Node           *DHT.Node
	availablePeers [maxPeerNum]PeerInfo

	//TODO: change to []PieceInfo
	pieces         [maxPieceNum]PieceInfo

	totalPieces int
	file        *os.File
	server      *rpc.Server
	Self        PeerInfo
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
	tmp := t.Size()
	for i := 0; i < this.totalPieces; i++ {
		this.pieces[i].have = true
		this.pieces[i].hash = GetPieceHash(this.file, i)
		if tmp >= pieceSize {
			this.pieces[i].length = pieceSize
		} else {
			this.pieces[i].length = int(tmp)
		}
		tmp -= pieceSize
		this.pieces[i].isLast = false
	}
	this.pieces[this.totalPieces-1].isLast = true

	this.Node.Run()
	this.Node.Put(hash, encodePeer(this.Self.addr))
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
	this.Self = PeerInfo{DHT.GetLocalAddress() + ":" + strconv.Itoa(port2)}
	this.startServer()
	this.Node.Run()
	if other != "" {
		this.Node.Join(other)
	}
}

func (this *Peer) nextToDownload() int {
	for i := 0; i < this.totalPieces; i++ {
		if this.pieces[i].have == false {
			/*
				TODO: choose rarest peer
			*/
			return i
		}
	}
	return 0
}

func (this *Peer) CheckPiece(pieceNum *int, reply *PieceInfo) error {
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
		var reply PieceInfo
		err = client.Call("Peer.GetPiece", &pieceNum, &reply)
		client.Close()
		if err != nil {
			log.Fatal("Download Failed: ", err)
		}
		if reply.have {
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
	//verify content
	if GetPieceHash(this.file, pieceNum).Cmp(this.pieces[pieceNum].hash) == 0 {
		this.pieces[pieceNum].have = true
	} else {
		log.Fatal("Download", pieceNum, "failed")
	}
}

/* Get Node Info (torrent.info) */
func (this *Peer) getTorrentInfo() {
	for i := 0; i < maxPieceNum; i++ {
		for _, p := range this.availablePeers {
			client, err := this.Connect(p)
			if err != nil {
				fmt.Printf("Ping %s Failed: ", p.addr)
				continue
			}
			var reply PieceInfo
			err = client.Call("Peer.GetPiece", &i, &reply)
			if err != nil {
				continue
			}
			_ = client.Close()
			if reply.have == true {
				this.pieces[i] = reply
				this.pieces[i].have = false
				break
			}
		}
		if this.pieces[i].isLast {
			this.totalPieces = i
			break
		}
	}
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

	/* Get Node Info (torrent.info) */
	this.getTorrentInfo()

	/* allocate file */
	this.file, _ = os.Create(fileName)
	if this.file == nil {
		log.Fatal("Can't create file")
	}
	for i := 0; i < this.totalPieces; i++ {
		_, err := this.file.Write(make([]byte, pieceSize))
		if err != nil {
			log.Fatal("Can't allocate file")
		}
	}

	/* Start download */
	/*
		TODO: Append self to the list
	*/
	copy(this.availablePeers[:], parsePeer(value))

	for len(this.pieces) != this.totalPieces {
		cur := this.nextToDownload()
		this.download(cur)
	}
	this.stripFile()

	/* Verify the entire content */
	curHash := GetFileHash(this.file)
	if curHash.String() != hash {
		log.Fatal("Download Failed")
		return false
	}
	fmt.Println("Download successful")
	return true
}
