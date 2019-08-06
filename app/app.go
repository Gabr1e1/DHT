package main

import (
	"./torrent-Kad"
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func main() {
	reader := bufio.NewReader(os.Stdin)
	peer := new(torrent_Kad.Peer)

	for {
		cmd, _, _ := reader.ReadLine()
		words := strings.Fields(string(cmd))
		if len(words) == 0 {
			continue
		}
		switch string(words[0]) {
		case "run":
			port, _ := strconv.Atoi(words[2])
			peer.Run(words[1], port)
		case "PublishFile":
			link := peer.PublishFile(words[1])
			fmt.Println("Magnet link generated: ", link)
		case "PublishFolder":
			link := peer.PublishFolder(words[1])
			fmt.Println("Magnet link generated: ", link)
		case "Download":
			ok := peer.Download(words[1])
			if ok {
				fmt.Println("Download success")
			} else {
				fmt.Println("Download failed")
			}
		default:
			fmt.Println("WRONG COMMAND, GO EAT SHIT")
		}
	}
}
