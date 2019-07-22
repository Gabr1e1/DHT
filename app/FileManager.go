package app

import (
	"crypto/sha1"
	"io"
	"log"
	"math/big"
	"os"
)

const pieceSize = 128 * 1024

func GetFileHash(file *os.File) *big.Int {
	h := sha1.New()
	if _, err := io.Copy(h, file); err != nil {
		log.Fatal("Can't hash file")
	}
	hRes := h.Sum(nil)
	var hash big.Int
	hash.SetBytes(hRes)
	return &hash
}

func readFile(file *os.File, pieceNum int) []byte {
	_, err := file.Seek(int64(pieceNum*pieceSize), 0)
	if err != nil {
		log.Fatal("Can't read file")
	}
	ret := make([]byte, pieceSize)
	_, err = file.Read(ret)
	if err != nil {
		log.Fatal("Can't read file")
	}
	return ret
}

func writeToFile(file *os.File, data []byte) error {
	_, err := file.Write(data)
	_ = file.Sync()
	return err
}

func (this *Peer) stripFile() {
	var t []byte
	var trunc = 0
	t = readFile(this.file, this.totalPieces)
	for i := len(t) - 1; i >= 0; i-- {
		if t[i] != 0 {
			trunc = i
			break
		}
	}
	_ = this.file.Truncate(int64((this.totalPieces-1)*pieceSize + trunc))
}
