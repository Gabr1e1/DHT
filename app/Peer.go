package app

import (
	"../src"
	"fmt"
	"log"
	"math"
	"math/big"
	"math/rand"
	"net"
	"net/rpc"
	"os"
	"strconv"
	"sync"
)

const maxPeerNum = 100
const maxPieceNum = 65536

type PeerInfo struct {
	addr string
}

type PieceInfo struct {
	Have   bool
	Hash   *big.Int
	Length int
	IsLast bool
}

type Peer struct {
	Node           *DHT.Node
	availablePeers []PeerInfo
	Pieces         []PieceInfo
	totalPieces    int
	file           *os.File
	server         *rpc.Server
	Self           PeerInfo
	wg             *sync.WaitGroup
	fileLock       sync.Mutex
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

	var hash = decToHex(this.GetFileHash())
	var link = fmt.Sprint("magnet:?Hash=" + hash + "?fn=" + fileName)
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
		var cur PieceInfo
		cur.Have = true
		cur.Hash = this.GetPieceHash(i)
		if tmp >= pieceSize {
			cur.Length = pieceSize
		} else {
			cur.Length = int(tmp)
		}
		tmp -= pieceSize
		cur.IsLast = false
		this.Pieces = append(this.Pieces, cur)
	}
	this.Pieces[this.totalPieces-1].IsLast = true

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

func (this *Peer) Run(port1 int, port2 int, other string, wg *sync.WaitGroup) {
	this.wg = wg
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

func (this *Peer) calcPiece(pieceNum int) int {
	return 1

	cnt := 0
	for _, p := range this.availablePeers {
		client, err := this.Connect(p)
		if err != nil {
			continue
		}
		var tmp bool
		err = client.Call("Peer.CheckPiece", &pieceNum, &tmp)
		_ = client.Close()
		if err != nil {
			continue
		}
		if tmp {
			cnt++
		}
	}
	return cnt
}

func (this *Peer) nextToDownload() int {
	min := maxPeerNum + 1
	mini := 0
	for i := 0; i < len(this.Pieces); i++ {
		if this.Pieces[i].Have == false {
			// choose rarest piece
			k := this.calcPiece(i)
			if k <= min {
				min = k
				mini = i
			}
		}
	}
	return mini
}

func (this *Peer) GetPiece(pieceNum *int, reply *PieceInfo) error {
	*reply = this.Pieces[*pieceNum]
	return nil
}

func (this *Peer) getBestPeer(curPeer []PeerInfo) PeerInfo {
	/*
		TODO: use tit-tat strategy
	*/
	return curPeer[rand.Intn(len(curPeer))]
}

func (this *Peer) download(pieceNum int) {
	/* Decide from which peer to download */
	var curPeer []PeerInfo
	for i := 0; i < len(this.availablePeers); i++ {
		if this.availablePeers[i].addr == this.Self.addr {
			continue
		}
		client, err := this.Connect(this.availablePeers[i])
		if err != nil {
			log.Fatal("Download Failed: ", err)
		}
		var reply PieceInfo
		err = client.Call("Peer.GetPiece", &pieceNum, &reply)
		_ = client.Close()
		if err != nil {
			log.Fatal("Download Failed: ", err)
		}
		if reply.Have {
			curPeer = append(curPeer, this.availablePeers[i])
		}
	}

	p := this.getBestPeer(curPeer)
	client, err := this.Connect(p)
	if err != nil {
		log.Fatal("Download Failed: ", err)
	}
	req := Request{this.Self.addr, pieceNum}
	fmt.Println(this.Self.addr, " Downloading piece", pieceNum, "from ", p)
	var data []byte
	err = client.Call("Peer.UploadData", &req, &data)
	_ = client.Close()
	if err != nil {
		log.Fatal("Download Failed: ", err)
	}

	err = this.writeToFile(data, pieceNum)

	if err != nil {
		log.Fatal("Can't write to file: ", err)
	}
	//verify content
	if this.GetPieceHash(pieceNum).Cmp(this.Pieces[pieceNum].Hash) == 0 {
		this.Pieces[pieceNum].Have = true
		this.totalPieces++
	} else {
		fmt.Println(this.GetPieceHash(pieceNum).String(), this.Pieces[pieceNum].Hash.String())
		log.Fatal("Download ", pieceNum, "failed, hash don't match")
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
			_ = client.Close()
			if err != nil {
				continue
			}
			_ = client.Close()
			if reply.Have == true {
				this.Pieces = append(this.Pieces, reply)
				this.Pieces[i].Have = false
				break
			}
		}
		if this.Pieces[i].IsLast {
			break
		}
	}
}

func (this *Peer) Download(link string, name string) bool {
	defer this.wg.Done()

	ok, hash, fileName := parseMagnetLink(link)
	if name != "" {
		fileName = name
	}
	fileName = fileName + ".download"
	if !ok {
		log.Fatal("Not a valid link")
	}
	success, value := this.Node.Get(hash)
	if !success {
		log.Fatal(this.Self, " Can't get value from dht")
	}

	/* Get Node Info (torrent.info) */
	this.availablePeers = parsePeer(value)
	this.getTorrentInfo()

	/* allocate file */
	this.file, _ = os.Create(fileName)
	if this.file == nil {
		log.Fatal("Can't create file")
	}
	t := make([]byte, pieceSize)
	for i := 0; i < this.totalPieces; i++ {
		_, err := this.file.Write(t)
		if err != nil {
			log.Fatal("Can't allocate file")
		}
	}
	_ = this.file.Sync()

	/* Start download */
	this.Node.AppendTo(hash, encodePeer(this.Self.addr))
	for len(this.Pieces) != this.totalPieces {
		cur := this.nextToDownload()
		this.download(cur)
	}

	/* Verify the entire content */
	this.fileLock.Lock()
	_ = this.file.Close()
	this.file, _ = os.Open(fileName)
	this.fileLock.Unlock()
	curHash := this.GetFileHash()
	if decToHex(curHash) != hash {
		fmt.Println("sha1 of downloaded file is", decToHex(curHash))
		log.Fatal(this.file.Name(), " Download Failed")
		return false
	}
	fmt.Println("Download successful")
	return true
}
