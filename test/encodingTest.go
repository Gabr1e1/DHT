package main

import (
	"../app/torrent-Kad"
	"fmt"
	"log"
	"os"
)

func main() {
	file, err := os.Open("src.torrent")
	if err != nil {
		log.Fatal(err)
	}

	str := make([]byte, 2000)
	l, _ := file.Read(str)
	str = str[0:l]

	t := torrent_Kad.Parse(string(str))
	fmt.Println(t)
}
