package main

import (
	"../app"
	"fmt"
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

	wg.Add(5)
	for i := 0; i < 5; i++ {
		var t app.Peer
		t.Run(port, port+1, seed.Node.Info.IPAddr, &wg)
		go t.Download(link, "lol"+strconv.Itoa(i)+".exe")
		time.Sleep(3 * time.Second)
		port += 2
	}
	wg.Wait()
	fmt.Println(time.Now())
}
