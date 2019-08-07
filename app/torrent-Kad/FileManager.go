package torrent_Kad

import (
	"fmt"
	"log"
	"os"
	"sync"
)

type FileInfo struct {
	Torrent []byte
	dec     map[interface{}]interface{}

	Pieces   IntSet
	File     *os.File
	FileLock sync.Mutex

	folderName    string
	PeerInfo      []PeerInfo
	isDownloading map[int]bool
	lock          sync.Mutex
}

func parseDir(path []interface{}) (string, string) {
	dir, fileName := "", ""
	for i := 0; i < len(path)-1; i++ {
		dir += path[i].(string) + "/"
	}
	fileName = path[len(path)-1].(string)
	return dir, fileName
}

func (this *FileInfo) Walk(deal func(path string, size int) error) error {
	//fmt.Println(this.dec["files"])
	for _, i := range this.dec["files"].([]interface{}) {
		dir, fileName := parseDir(i.(map[interface{}]interface{})["path"].([]interface{}))
		//fmt.Println(this.folderName+"/"+dir+fileName, i.(map[interface{}]interface{})["length"].(int))
		err := deal(this.folderName+"/"+dir+fileName, i.(map[interface{}]interface{})["length"].(int))
		if err != nil {
			return err
		}
	}
	return nil
}

func (this *FileInfo) GetFileInfo(index int, length int) []byte {
	this.FileLock.Lock()
	defer this.FileLock.Unlock()
	ret := make([]byte, pieceSize)

	if this.File != nil { //case 1: single file
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
		ret = make([]byte, 0)
		cur := 0
		err := this.Walk(
			func(path string, size int) error {
				if len(ret) > length {
					return nil
				}
				if (cur <= index*pieceSize && cur+size > index*pieceSize) || (cur >= index*pieceSize && cur < (index+1)*pieceSize) {
					file, err := os.OpenFile(path, os.O_RDONLY, 0666)
					defer file.Close()
					if err != nil {
						return err
					}
					start := Max(0, index*pieceSize-cur)
					t := make([]byte, pieceSize)
					_, _ = file.Seek(int64(start), 0)
					l, err := file.Read(t)
					if err != nil {
						log.Fatal(file.Name()+" Can't read at ", err)
					}
					fmt.Println("Read: ", start, l, size, length)

					t = t[:Min(length, l)]
					ret = append(ret, t...)
				}
				cur += size
				return nil
			})
		if err != nil {
			log.Fatal("Can't read file3: ", this.folderName, err)
		}
		return ret[0:Min(length, len(ret))]
	}
}

func (this *FileInfo) writeToFile(index int, data []byte) error {
	this.FileLock.Lock()
	defer this.FileLock.Unlock()

	if this.File != nil {
		_, err := this.File.Seek(int64(index*pieceSize), 0)
		if err != nil {
			return err
		}
		_, err = this.File.Write(data)
		_ = this.File.Sync()
		return err
	} else {
		cur := 0
		err := this.Walk(
			func(path string, size int) error {
				if len(data) == 0 {
					return nil
				}

				if (cur <= index*pieceSize && cur+size > index*pieceSize) || (cur >= index*pieceSize && cur < (index+1)*pieceSize) {
					file, err := os.OpenFile(path, os.O_RDWR, 0666)
					defer file.Close()
					if err != nil {
						fmt.Println(err)
						return err
					}
					start := Max(0, index*pieceSize-cur)
					_, _ = file.Seek(int64(start), 0)
					l, _ := file.Write(data[0:Min(len(data), size-start)])

					fmt.Println("WRITE", file.Name(), start, l, len(data))
					data = data[l:]
				}
				cur += size
				return nil
			})
		if err != nil {
			log.Fatal("Can't read file3: ", this.folderName, err)
		}
		return nil
	}
}
