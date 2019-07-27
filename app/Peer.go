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
	"sort"
	"strconv"
	"sync"
)

const maxPieceNum = 65536
const maxConcurrentThreads = 10

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
	order          []int
	orderLock      sync.Mutex
	totalPieces    int

	file     *os.File
	server   *rpc.Server
	Self     PeerInfo
	wg       *sync.WaitGroup
	fileLock sync.Mutex
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

	//bigger pieceSize for big file
	if t.Size() > 1024*1024*1024 {
		pieceSize = 1024 * 1024
	}

	this.totalPieces = int(math.Ceil(float64(t.Size()) / float64(pieceSize)))
	tmp := t.Size()
	for i := 0; i < this.totalPieces; i++ {
		var cur PieceInfo
		cur.Have = true
		cur.Hash = this.GetPieceHash(i)
		if tmp >= int64(pieceSize) {
			cur.Length = pieceSize
		} else {
			cur.Length = int(tmp)
		}
		tmp -= int64(pieceSize)
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

func (this *Peer) nextToDownload() int {
	this.orderLock.Lock()
	if len(this.order) == 0 {
		this.orderLock.Unlock()
		return -1
	}

	ret := this.order[0]
	this.order = this.order[1:]
	this.orderLock.Unlock()
	return ret
}

func (this *Peer) GetPiece(pieceNum *int, reply *PieceInfo) error {
	*reply = this.Pieces[*pieceNum]
	return nil
}

func (this *Peer) getBestPeer(curPeer []PeerInfo) PeerInfo {
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

func sortByValue(m map[int]int) []int {
	var keys []int
	for key := range m {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		return m[keys[i]] < m[keys[j]]
	})
	return keys
}

/* Get Node Info (torrent.info) */
func (this *Peer) getTorrentInfo() {
	var a map[int]int
	a = make(map[int]int)

	for i := 0; i < maxPieceNum; i++ {
		cur := 0
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
				if cur == 0 {
					this.Pieces = append(this.Pieces, reply)
					this.Pieces[i].Have = false
				}
				cur++
			}
		}
		a[i] = cur
		if this.Pieces[i].IsLast {
			break
		}
	}
	this.order = sortByValue(a)
	fmt.Println(this.Self, "ORDER:", this.order)
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
	_ = this.file.Truncate(int64(this.totalPieces * pieceSize))
	_ = this.file.Sync()

	/* Start download */
	this.Node.AppendTo(hash, encodePeer(this.Self.addr))
	var wg sync.WaitGroup
	wg.Add(maxConcurrentThreads)
	for i := 0; i < maxConcurrentThreads; i++ {
		go func() {
			defer wg.Done()
			for {
				t := this.nextToDownload()
				if t == -1 {
					break
				}
				this.download(t)
			}
		}()
	}
	wg.Wait()

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
	fmt.Println(this.file.Name(), "Download successful")
	return true
}
