package torrent_Kad

import (
	"../../src/Chord"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

const pieceSize = 256 * 1024

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
		t := int64(pieceSize - len(*data))
		fmt.Println("HASH", file.Name(), t)
		*data = append(*data, readFile(file, t)...)
		size -= t
		*str += DHT.GetByteHash(string(*data))
		*data = make([]byte, 0)
	}
	if size != 0 && size+int64(len(*data)) < int64(pieceSize) {
		*data = append(*data, readFile(file, size)...)
	}
}

func EncodeStr(x string) []byte {
	ret := fmt.Sprintf("%d", len(x)) + ":" + x
	return []byte(ret)
}

func EncodeNum(x int) []byte {
	ret := fmt.Sprintf("i%de", x)
	return []byte(ret)
}

func EncodeList(list []interface{}) []byte {
	ret := "l"
	for _, i := range list {
		ret += string(Encode(i))
	}
	ret += "e"
	return []byte(ret)
}

func EncodeMap(m map[interface{}]interface{}) []byte {
	ret := "d"
	for k, v := range m {
		ret += string(Encode(k)) + string(Encode(v))
	}
	ret += "e"
	return []byte(ret)
}

func Encode(enc interface{}) []byte {
	switch reflect.TypeOf(enc).Kind() {
	case reflect.Int:
		return EncodeNum(enc.(int))
	case reflect.Int64:
		return EncodeNum(int(enc.(int64)))
	case reflect.Slice:
		list := make([]interface{}, 0)
		t := reflect.ValueOf(enc)
		for i := 0; i < t.Len(); i++ {
			list = append(list, t.Index(i).Interface())
		}
		return EncodeList(list)
	case reflect.Map:
		return EncodeMap(enc.(map[interface{}]interface{}))
	default:
		return EncodeStr(enc.(string))
	}
}

func EncodeFolder(folderName string) ([]byte, string) {
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
			if info.Size() == 0 || len(path) < len(folderName)+1 || info.IsDir() {
				return nil
			}
			fmt.Println(path, info.Size())

			cur["length"] = info.Size()
			//fmt.Println("File: ", path[len(folderName)+1:])
			cur["path"] = strings.Split(path[len(folderName)+1:], "\\")

			files = append(files, cur)
			addHash(&pieces, &data, path)

			return nil
		})
	if err != nil {
		fmt.Println(err)
		return nil, ""
	}
	enc["files"] = files
	enc["piece length"] = 262144

	if len(data) != 0 {
		pieces += DHT.GetByteHash(string(data))
	}
	enc["pieces"] = pieces
	return Encode(enc), pieces
}

func EncodeSingleFile(fileName string) ([]byte, string) {
	enc := make(map[interface{}]interface{})
	enc["name"] = fileName

	file, err := os.Open(fileName)
	if err != nil {
		fmt.Println("Can't open File")
		return nil, ""
	}
	info, _ := file.Stat()
	enc["length"] = info.Size()
	enc["piece length"] = 262144
	pieces := ""
	data := make([]byte, 0)

	addHash(&pieces, &data, fileName)
	if len(data) != 0 {
		pieces += DHT.GetByteHash(string(data))
	}
	enc["pieces"] = pieces
	return Encode(enc), pieces
}
