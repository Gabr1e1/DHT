package main

import (
	"../app"
	"sync"
	"time"
)

func main() {
	var seed app.Peer
	link := seed.Publish("lol.exe", 4509, 2000, "")

	var wg sync.WaitGroup

	var downloader app.Peer
	downloader.Run(1234, 5678, seed.Node.Info.IPAddr, &wg)
	time.Sleep(1.0 * time.Second)

	var third app.Peer
	third.Run(4512, 4513, seed.Node.Info.IPAddr, &wg)
	time.Sleep(1.0 * time.Second)
	var fourth app.Peer
	fourth.Run(4514, 4515, seed.Node.Info.IPAddr, &wg)
	time.Sleep(3.0 * time.Second)

	wg.Add(1)
	go downloader.Download(link, "")
	wg.Wait()

	wg.Add(2)
	go third.Download(link, "lol2.exe")
	go fourth.Download(link, "lol3.exe")
	wg.Wait()
}
