package app

import (
	"crypto/sha1"
	"io"
	"log"
	"math/big"
)

const pieceSize = 128 * 1024

func (this *Peer) GetFileHash() *big.Int {
	this.fileLock.Lock()
	h := sha1.New()
	if _, err := io.Copy(h, this.file); err != nil {
		log.Fatal("Can't Hash file")
	}
	this.fileLock.Unlock()
	hRes := h.Sum(nil)
	var hash big.Int
	hash.SetBytes(hRes)
	return &hash
}

func (this *Peer) GetPieceHash(pieceNum int) *big.Int {
	t := this.readFile(pieceNum)
	h := sha1.New()
	h.Write(t)
	hRes := h.Sum(nil)
	var hash big.Int
	hash.SetBytes(hRes)
	return &hash
}

func (this *Peer) readFile(pieceNum int) []byte {
	this.fileLock.Lock()
	_, err := this.file.Seek(int64(pieceNum*pieceSize), 0)
	if err != nil {
		log.Fatal("Can't read file1: ", this.file.Name(), " ", pieceNum, " ", err)
	}
	ret := make([]byte, pieceSize)
	_, err = this.file.Read(ret)
	if err != nil {
		log.Fatal("Can't read file2: ", this.file.Name(), " ", pieceNum, " ", err)
	}
	this.fileLock.Unlock()
	return ret
}

func (this *Peer) writeToFile(data []byte, pieceNum int) error {
	this.fileLock.Lock()
	_, err := this.file.Seek(int64(pieceNum*pieceSize), 0)
	_, err = this.file.Write(data[:this.Pieces[pieceNum].Length])
	_ = this.file.Sync()
	this.fileLock.Unlock()
	return err
}
