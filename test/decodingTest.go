package main

import (
	"../app/torrent-Kad"
	"fmt"
)

func main() {
	//t := make([]interface{}, 2)
	//t[0] = "dddd"
	//t[1] = "dfdf"
	//
	//m := make(map[interface{}]interface{})
	//m["abc"] = t
	//fmt.Println(string(torrent_Kad.EncodeMap(m)))

	//t := torrent_Kad.EncodeFolder("src")
	t := torrent_Kad.EncodeSingleFile("src.7z")
	fmt.Println(string(t))
	fmt.Println(torrent_Kad.Decode(string(t)))
}
