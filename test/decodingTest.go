package main

import "fmt"

func main() {
	t := make([]byte, 1)
	fmt.Println(len(t), cap(t))
	t = append(t, 1, 2, 3)
	fmt.Println(len(t), cap(t))

	//torrent_Kad.EncodeFolder("src")
}
