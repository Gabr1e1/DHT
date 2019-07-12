package main

import (
	"../src"
	"time"
)

func main() {
	var n1 DHT.Node
	n1.Create("127.0.0.1:15000")
	n1.Run()

	var n2 DHT.Node
	n2.Create("127.0.0.1:15001")
	n2.Run()

	for i := 1; i < 1000; i++ {
		go n1.Connect(n2.Info)
	}
	time.Sleep(10 * time.Second)
}
