package main

import (
	"../app"
	"strconv"
	"sync"
	"time"
)

func main() {
	var seed app.Peer
	port := 4000
	link := seed.Publish("lol.exe", port, port+1, "")
	port += 2

	var wg sync.WaitGroup
	wg.Add(20)
	for i := 0; i < 20; i++ {
		var t app.Peer
		t.Run(port, port+1, seed.Node.Info.IPAddr, &wg)
		go t.Download(link, "lol"+strconv.Itoa(i)+".exe")
		time.Sleep(5 * time.Second)
		port += 2
	}
	wg.Wait()
}
