package torrent_Kad

import (
	"../../src/Chord"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var pieceSize = 256 * 1024

type BEncoding struct {
	data   map[interface{}]interface{}
	reader *strings.Reader
}

func readFile(file *os.File, size int64) []byte { //read from the last position
	t := make([]byte, size)
	l, _ := file.Read(t)
	return t[0:l]
}

func addHash(str *string, data *[]byte, path string) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	stat, _ := file.Stat()
	size := stat.Size()
	//fmt.Println(size)

	for size+int64(len(*data)) >= int64(pieceSize) {
		*data = append(*data, readFile(file, int64(pieceSize-len(*data)))...)
		size -= int64(pieceSize - len(*data))
		*str += DHT.GetByteHash(string(*data))
		*data = (*data)[:0] //empty data but keep allocated memory
	}
	if size != 0 && size+int64(len(*data)) < int64(pieceSize) {
		*data = append(*data, readFile(file, size)...)
	}
}

func EncodeNum(x int) {

}
func Encode(enc interface{}) []byte {

}

func EncodeFolder(folderName string) []byte {
	enc := make(map[interface{}]interface{})
	enc["name"] = folderName

	var files []map[interface{}]interface{}
	var pieces string
	data := make([]byte, pieceSize)[:0]

	err := filepath.Walk(folderName,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			cur := make(map[interface{}]interface{})
			if info.Size() == 0 || len(path) < len(folderName)+1 {
				return nil
			}
			cur["length"] = info.Size()
			cur["path"] = strings.ReplaceAll(path[len(folderName)+1:], "\\", " ")
			files = append(files, cur)
			addHash(&pieces, &data, path)

			return nil
		})
	if err != nil {
		fmt.Println(err)
		return nil
	}
	enc["files"] = files
	enc["piece length"] = 262144

	if len(data) != 0 {
		pieces += DHT.GetByteHash(string(data))
	}
	enc["pieces"] = pieces
	return Encode(enc)
}
