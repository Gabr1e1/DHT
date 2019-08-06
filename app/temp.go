package main

import (
	"./torrent-Kad"
	"fmt"
	"os"
)

func main() {
	file, _ := os.Open("IdeaProjects.torrent")
	t := make([]byte, 256*1024)
	l, _ := file.Read(t)
	fmt.Println(torrent_Kad.Decode(string(t[0:l])))
}
