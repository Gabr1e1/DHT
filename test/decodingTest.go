package main

import (
	"../app/torrent-Kad"
	"fmt"
)

func main() {
	t := torrent_Kad.EncodeFolder("src")
	fmt.Println(string(t))
}
