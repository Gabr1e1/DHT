package torrent_Kad

import (
	"../../src/Chord"
	"../../src/Kademlia"
	"errors"
	"fmt"
	"github.com/cheggaaa/pb"
	"log"
	"math/rand"
	"net/rpc"
	"os"
	"strconv"
	"sync"
)

const maxConcurrentThread = 3

type PeerInfo struct {
	Addr   string
	pieces IntSet
}

type Peer struct {
	Node     *Kademlia.Node
	FileStat map[string]FileInfo
	server   *rpc.Server
	addr     string
	lock     sync.Mutex
}

func (this *Peer) Run(addr string, port int) {
	this.Node = new(Kademlia.Node)
	this.Node.Create(addr)
	this.Node.Run(port)

	_ = this.Node.Server.Register(this)
	this.addr = addr
	go this.Node.Server.Accept(this.Node.Listener)

	this.FileStat = make(map[string]FileInfo)
}

func (this *Peer) PublishFile(fileName string) string {
	file, err := os.OpenFile(fileName, os.O_RDWR, 0777)
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

	dec := Decode(string(this.FileStat[infoHash].Torrent)).(map[interface{}]interface{})
	t := this.FileStat[infoHash]
	t.dec = dec
	this.FileStat[infoHash] = t

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

	dec := Decode(string(this.FileStat[infoHash].Torrent)).(map[interface{}]interface{})
	t := this.FileStat[infoHash]
	t.dec = dec
	this.FileStat[infoHash] = t

	link := "magnet:?xt=urn:btih:" + infoHash + "&dn=" + folderName + "&tr=" + this.addr
	return link
}

func (this *Peer) initDownload(magnetLink string) (string, error) {
	infoHash, _, tracker := parseMagnet(magnetLink)
	this.Node.Join(tracker)
	//fmt.Println(infoHash)

	ok, peerList := this.Node.Get(infoHash)
	if !ok {
		return "", errors.New("can't find corresponding Torrent File")
	}

	fmt.Println("PeerList: ", peerList)

	var ret string
	var err error
	for _, peer := range peerList {
		client, err := this.Connect(peer)
		if err != nil {
			continue
		}
		torrent := make([]byte, maxTorrentSize)[:0]
		err = client.Call("Peer.GetTorrentFile", &infoHash, &torrent)
		_ = client.Close()
		if err != nil {
			fmt.Println(peer, err)
			continue
		}
		if len(torrent) > 0 {
			t := this.FileStat[infoHash]
			t.Torrent = torrent
			this.FileStat[infoHash] = t
			ret, err = infoHash, nil
			break
		}
	}

	peerInfo := make([]PeerInfo, 0)
	for _, addr := range peerList {
		client, err := this.Connect(addr)
		if err != nil {
			continue
		}
		curSet := make(IntSet)
		err = client.Call("Peer.GetPieceStatus", &infoHash, &curSet)
		if _, ok := curSet[-1]; ok { //Have all pieces
			dec := Decode(string(this.FileStat[infoHash].Torrent)).(map[interface{}]interface{})
			num := len(dec["pieces"].(string)) / 20
			for i := 0; i < num; i++ {
				curSet[i] = struct{}{}
			}
			delete(curSet, -1)
		}

		//for k := range curSet {
		//	fmt.Println("Cur Piece: ", k)
		//}
		_ = client.Close()
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

	if len(ret) > 0 {
		return ret, err
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

func (this *Peer) verify(infoHash string, pieceNum int, curPiece []byte, dec map[interface{}]interface{}) bool {
	a := []byte(DHT.GetByteHash(string(curPiece)))
	b := []byte(dec["pieces"].(string)[pieceNum*20:(pieceNum+1)*20])
	if len(a) != len(b) {
		fmt.Println("Downloaded", pieceNum, "len: ", len(curPiece))
		fmt.Println("Hash of downloaded file: \n", []byte(DHT.GetByteHash(string(curPiece))))
		fmt.Println("Expected hash: \n", []byte(dec["pieces"].(string)[pieceNum*20:(pieceNum+1)*20]))
		return false
	}
	for i := 0; i < len(a); i++ {
		if a[i] != b[i] {
			fmt.Println("Downloaded", pieceNum, "len: ", len(curPiece))
			fmt.Println("Hash of downloaded file: \n", []byte(DHT.GetByteHash(string(curPiece))))
			fmt.Println("Expected hash: \n", []byte(dec["pieces"].(string)[pieceNum*20:(pieceNum+1)*20]))
			return false
		}
	}
	return true
}

func (this *Peer) download(infoHash string, pieceNum int, dec map[interface{}]interface{}) error {
	//fmt.Println("Downloading piece: ", pieceNum)
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
	_ = client.Close()
	if err != nil {
		return err
	}
	t := this.FileStat[infoHash]

	ch := make(chan error)
	go func() {
		err = t.writeToFile(pieceNum, curPiece)
		if err != nil {
			ch <- err
		}
		this.lock.Lock()
		defer this.lock.Unlock()

		if this.verify(infoHash, pieceNum, curPiece, dec) {
			t := this.FileStat[infoHash]
			t.isDownloading[pieceNum] = false
			t.Pieces[pieceNum] = struct{}{}
			this.FileStat[infoHash] = t
			ch <- nil
		} else {
			t := this.FileStat[infoHash]
			t.isDownloading[pieceNum] = false
			this.FileStat[infoHash] = t
			ch <- errors.New("wrong data")
		}
	}()

	err = <-ch
	return err
}

func (this *Peer) allocate(infoHash string, dec map[interface{}]interface{}) {
	if _, ok := dec["length"]; ok {
		/* allocate file */
		file, _ := os.Create(dec["name"].(string))
		if file == nil {
			log.Fatal("Can't create file")
		}
		_ = file.Truncate(int64(dec["length"].(int)))
		_ = file.Sync()
		t := this.FileStat[infoHash]
		t.Pieces = make(IntSet)
		t.File = file
		this.FileStat[infoHash] = t

	} else {
		t := this.FileStat[infoHash]
		t.folderName = dec["name"].(string)
		t.Pieces = make(IntSet)
		this.FileStat[infoHash] = t

		files := dec["files"].([]interface{})
		for _, i := range files {
			curFile := i.(map[interface{}]interface{})
			dir, fileName := parseDir(curFile["path"].([]interface{}))
			//fmt.Println(curFile["path"].([]interface{}))
			_ = os.MkdirAll(dec["name"].(string)+"/"+dir, os.ModePerm)
			/* allocate file */
			file, _ := os.Create(dec["name"].(string) + "/" + dir + fileName)
			if file == nil {
				log.Fatal("Can't create file ", dec["name"].(string)+"/"+dir+fileName, dir, fileName)
			}
			num := curFile["length"].(int) / pieceSize
			if curFile["length"].(int)%pieceSize != 0 {
				num++
			}
			_ = file.Truncate(int64(curFile["length"].(int)))
			_ = file.Close()
		}
	}
}

func (this *Peer) truncate(infoHash string, dec map[interface{}]interface{}) {
	if _, ok := dec["length"]; ok {
		_ = this.FileStat[infoHash].File.Truncate(int64(dec["length"].(int)))
		_ = this.FileStat[infoHash].File.Close()
	} else {
		files := dec["files"].([]interface{})
		for _, i := range files {
			curFile := i.(map[interface{}]interface{})
			dir, fileName := parseDir(curFile["path"].([]interface{}))
			file, _ := os.OpenFile(dec["name"].(string)+"/"+dir+fileName, os.O_RDWR, 0777)
			err := file.Truncate(int64(curFile["length"].(int)))
			if err != nil {
				log.Fatal("Can't truncate: ", err)
			}
			_ = file.Close()
		}
	}
}

func (this *Peer) getNextPiece(infoHash string, total int) int {
	this.lock.Lock()
	t := this.FileStat[infoHash]
	defer this.lock.Unlock()

	for i := 0; i < total; i++ {
		if _, ok := t.Pieces[i]; (!ok) && (!t.isDownloading[i]) {
			t.isDownloading[i] = true
			this.FileStat[infoHash] = t
			return i
		}
	}
	return -1
}

func (this *Peer) Download(magnetLink string) bool {
	infoHash, err := this.initDownload(magnetLink)
	if err != nil {
		fmt.Println("Download failed", err)
		return false
	}

	this.lock.Lock()
	dec := Decode(string(this.FileStat[infoHash].Torrent)).(map[interface{}]interface{})
	t := this.FileStat[infoHash]
	t.dec = dec
	t.isDownloading = make(map[int]bool)
	this.FileStat[infoHash] = t
	this.lock.Unlock()

	this.allocate(infoHash, dec)
	this.Node.Put(infoHash, this.addr)

	total := len(dec["pieces"].(string)) / 20
	bar := pb.StartNew(total)

	var wg sync.WaitGroup
	wg.Add(maxConcurrentThread)
	for i := 0; i < maxConcurrentThread; i++ {
		go func() {
			defer wg.Done()
			for {
				t := this.getNextPiece(infoHash, total)
				if t == -1 {
					break
				}
				err := this.download(infoHash, t, dec)
				if err != nil {
					fmt.Println("Piece", t, "download failed", err)
				} else {
					bar.Increment()
				}
			}
		}()
	}
	wg.Wait()
	bar.Finish()
	return true
}
