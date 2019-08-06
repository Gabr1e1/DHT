package torrent_Kad

import (
	"../../src/Chord"
	"../../src/Kademlia"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net/rpc"
	"os"
	"strconv"
)

type PeerInfo struct {
	Addr   string
	pieces IntSet
}

type Peer struct {
	Node     *Kademlia.Node
	joined   bool
	FileStat map[string]FileInfo
	server   *rpc.Server
	addr     string
}

func (this *Peer) Run(addr string, port int) {
	this.Node = new(Kademlia.Node)
	this.Node.Create(addr)
	this.Node.Run(port)
	this.joined = false

	_ = this.Node.Server.Register(this)
	this.addr = Kademlia.GetLocalAddress() + ":" + strconv.Itoa(port)
	go this.Node.Server.Accept(this.Node.Listener)

	this.FileStat = make(map[string]FileInfo)
}

func (this *Peer) PublishFile(fileName string) string {
	file, err := os.Open(fileName)
	if err != nil {
		fmt.Println("Can't open File")
		return ""
	}
	torrent, piece := EncodeSingleFile(file.Name())
	pieces := make(IntSet)
	for i := 0; i < len(piece)/20; i++ {
		pieces[i] = struct{}{}
	}

	cur := FileInfo{Torrent: torrent, File: file, Pieces: pieces}
	infoHash := fmt.Sprintf("%x", DHT.GetHash(string(torrent)))
	this.FileStat[infoHash] = cur

	this.Node.Put(infoHash, this.addr)

	link := "magnet:?xt=urn:btih:" + infoHash + "&dn=" + fileName + "&tr=" + this.addr
	return link
}

func (this *Peer) PublishFolder(folderName string) string {
	torrent, piece := EncodeFolder(folderName)
	pieces := make(IntSet)
	for i := 0; i < len(piece)/20; i++ {
		pieces[i] = struct{}{}
	}

	cur := FileInfo{Torrent: torrent, folderName: folderName, Pieces: pieces}
	infoHash := fmt.Sprintf("%x", DHT.GetHash(string(torrent)))
	this.FileStat[infoHash] = cur

	this.Node.Put(infoHash, this.addr)

	link := "magnet:?xt=urn:btih:" + infoHash + "&dn=" + folderName + "&tr=" + this.addr
	return link
}

func (this *Peer) initDownload(magnetLink string) (string, error) {
	infoHash, _, tracker := parseMagnet(magnetLink)
	if !this.joined {
		this.Node.Join(tracker)
		this.joined = true
	}

	fmt.Println(infoHash)

	ok, peerList := this.Node.Get(infoHash)
	if !ok {
		return "", errors.New("can't find corresponding Torrent File")
	}

	peerInfo := make([]PeerInfo, 0)
	for _, addr := range peerList {
		client, err := this.Connect(addr)
		if err != nil || addr == this.addr {
			continue
		}
		curSet := make(IntSet)
		err = client.Call("Peer.GetPieceStatus", &infoHash, &curSet)
		if err != nil {
			fmt.Println(addr, err)
			continue
		}
		curInfo := PeerInfo{Addr: addr, pieces: curSet}
		peerInfo = append(peerInfo, curInfo)
	}
	t := this.FileStat[infoHash]
	t.PeerInfo = peerInfo
	this.FileStat[infoHash] = t

	for _, peer := range this.FileStat[infoHash].PeerInfo {
		client, err := this.Connect(peer.Addr)
		if err != nil {
			continue
		}
		torrent := make([]byte, maxTorrentSize)[:0]
		err = client.Call("Peer.GetTorrentFile", &infoHash, &torrent)
		if err != nil {
			fmt.Println(peer.Addr, err)
			continue
		}
		if len(torrent) > 0 {
			t := this.FileStat[infoHash]
			t.Torrent = torrent
			this.FileStat[infoHash] = t
			return infoHash, nil
		}
	}
	return "", errors.New("can't find Torrent")
}

func (this *Peer) choosePeer(infoHash string, pieceNum int) (PeerInfo, error) {
	t := this.FileStat[infoHash].PeerInfo
	rand.Shuffle(len(t), func(i, j int) {
		t[i], t[j] = t[j], t[i]
	})
	for _, peer := range this.FileStat[infoHash].PeerInfo {
		if _, ok := peer.pieces[pieceNum]; ok {
			return peer, nil
		}
	}
	return PeerInfo{}, errors.New(infoHash + "can't find" + strconv.Itoa(pieceNum))
}

func (this *Peer) verify(infoHash string, pieceNum int) bool {
	return true
}

func (this *Peer) download(infoHash string, pieceNum int) error {
	peer, err := this.choosePeer(infoHash, pieceNum)
	if err != nil {
		return err
	}
	client, err := this.Connect(peer.Addr)
	if err != nil {
		return err
	}
	curPiece := make([]byte, pieceSize)[:0]
	err = client.Call("Peer.GetPiece", &TorrentRequest{infoHash, pieceNum, pieceSize}, &curPiece)
	if err != nil {
		return err
	}
	t := this.FileStat[infoHash]
	err = t.writeToFile(pieceNum, curPiece)
	if err != nil {
		return err
	}
	if this.verify(infoHash, pieceNum) {
		t := this.FileStat[infoHash]
		t.Pieces[pieceNum] = struct{}{}
		this.FileStat[infoHash] = t
		return nil
	} else {
		return errors.New("wrong data")
	}
}

func (this *Peer) allocate(infoHash string, dec map[interface{}]interface{}) {
	num := len(dec["pieces"].(string)) / 20
	if _, ok := dec["length"]; ok {
		/* allocate file */
		file, _ := os.Create(dec["name"].(string))
		if file == nil {
			log.Fatal("Can't create file")
		}
		_ = file.Truncate(int64(num * pieceSize))
		_ = file.Sync()
		t := this.FileStat[infoHash]
		t.Pieces = make(IntSet)
		t.File = file
		this.FileStat[infoHash] = t
	} else {
		files := dec["files"].([]interface{})
		for _, i := range files {
			curFile := i.(map[interface{}]interface{})
			dir, fileName := parseDir(curFile["path"].([]interface{}))
			_ = os.MkdirAll(dir, os.ModePerm)

			/* allocate file */
			file, _ := os.Create(dir + fileName)
			if file == nil {
				log.Fatal("Can't create file")
			}
			_ = file.Truncate(int64(num * pieceSize))
			_ = file.Sync()
		}
	}
}

func (this *Peer) Download(magnetLink string) bool {
	infoHash, err := this.initDownload(magnetLink)
	if err != nil {
		fmt.Println("Download failed", err)
		return false
	}

	dec := Decode(string(this.FileStat[infoHash].Torrent)).(map[interface{}]interface{})
	num := len(dec["pieces"].(string)) / 20
	this.allocate(infoHash, dec)
	this.Node.Put(infoHash, this.addr)

	for i := 0; i < num; i++ {
		err := this.download(infoHash, i)
		if err != nil {
			fmt.Println("Can't get piece", i, err)
			return false
		}
	}

	_ = this.FileStat[infoHash].File.Truncate(int64(dec["length"].(int)))
	_ = this.FileStat[infoHash].File.Close()

	return true
}
