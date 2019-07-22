package main

import (
	"../app"
)

func main() {
	var seed app.Peer
	link := seed.Publish(".emacs", 4509, 2000, "")

	var downloader app.Peer
	downloader.Run(1234, 5678, seed.Node.Info.IPAddr)
	downloader.Download(link)
}
