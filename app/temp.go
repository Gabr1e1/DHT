package main

import (
	"fmt"
	"os"
)

func main() {
	file, _ := os.Open("src/app/app_client.go")
	t := make([]byte, 256*1024)
	l, _ := file.Read(t)
	fmt.Println(l, t[0:l])
	//fmt.Println(torrent_Kad.Decode(string(t[0:l])))
}
