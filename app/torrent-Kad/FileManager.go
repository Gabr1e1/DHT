package torrent_Kad

import (
	"crypto/sha1"
	"fmt"
	"io"
	"log"
	"math/big"
	"os"
	"sync"
)

type FileInfo struct {
	Torrent []byte

	Pieces   IntSet
	File     *os.File
	FileLock sync.Mutex
	PeerInfo []PeerInfo
}

func (this *FileInfo) GetFileInfo(index int, length int) []byte {
	this.FileLock.Lock()
	defer this.FileLock.Unlock()
	_, err := this.File.Seek(int64(index*pieceSize), 0)
	if err != nil {
		log.Fatal("Can't read file1: ", this.File.Name(), " ", index, " ", err)
	}
	ret := make([]byte, pieceSize)
	_, err = this.File.Read(ret)
	if err != nil {
		log.Fatal("Can't read file2: ", this.File.Name(), " ", index, " ", err)
	}
	fmt.Println(index, len(ret))
	return ret
}

func (this *FileInfo) writeToFile(index int, data []byte) error {
	this.FileLock.Lock()
	defer this.FileLock.Unlock()
	_, err := this.File.Seek(int64(index*pieceSize), 0)
	if err != nil {
		return err
	}
	_, err = this.File.Write(data)
	_ = this.File.Sync()
	return err
}

func (this *FileInfo) GetFileHash() *big.Int {
	this.FileLock.Lock()
	defer this.FileLock.Unlock()
	h := sha1.New()
	if _, err := io.Copy(h, this.File); err != nil {
		this.FileLock.Unlock()
		log.Fatal("Can't Hash File")
	}
	hRes := h.Sum(nil)
	var hash big.Int
	hash.SetBytes(hRes)
	return &hash
}
