package torrent_Kad

import (
	"crypto/sha1"
	"fmt"
	"io"
	"log"
	"math/big"
	"os"
	"path/filepath"
	"sync"
)

type FileInfo struct {
	Torrent []byte

	Pieces   IntSet
	File     *os.File
	FileLock sync.Mutex

	folderName string

	PeerInfo []PeerInfo
}

func (this *FileInfo) GetFileInfo(index int, length int) []byte {
	this.FileLock.Lock()
	defer this.FileLock.Unlock()
	ret := make([]byte, pieceSize)

	if this.File == nil { //case 1: single file
		_, err := this.File.Seek(int64(index*pieceSize), 0)
		if err != nil {
			log.Fatal("Can't read file1: ", this.File.Name(), " ", index, " ", err)
		}
		_, err = this.File.Read(ret)
		if err != nil {
			log.Fatal("Can't read file2: ", this.File.Name(), " ", index, " ", err)
		}
		return ret
	} else { //case 2: folder
		cur := 0
		err := filepath.Walk(this.folderName,
			func(path string, info os.FileInfo, err error) error {
				if cur >= index*pieceSize && cur < index*pieceSize+length {

				}
				return nil
			})
		if err != nil {
			log.Fatal("Can't read file3: ", this.folderName, err)
		}
	}
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

func (this *FileInfo) GetFolderHash() *big.Int {
	h := sha1.New()
	err := filepath.Walk(this.folderName,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.Size() == 0 || len(path) < len(this.folderName)+1 || info.IsDir() {
				return nil
			}
			file, err := os.Open(path)
			defer file.Close()
			if err != nil {
				return nil
			}
			_, _ = io.Copy(h, file)
			fmt.Println(path, info.Size(), h.Size())
			return nil
		})
	if err != nil {
		return nil
	}

	hRes := h.Sum(nil)
	var hash big.Int
	hash.SetBytes(hRes)
	return &hash
}
