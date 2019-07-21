package app

import (
	"crypto/sha1"
	"io"
	"log"
	"math/big"
	"os"
)

const pieceSize = 128 * 1024

func getFileHash(file *os.File) *big.Int {
	defer file.Close()
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
