package main

import (
	"../src"
	"fmt"
	"time"
)

func main() {
	var n1 DHT.Node
	n1.Create("127.0.0.1:2000")
	n1.Run()
	n1.Put("aaa", "bbb")

	var n2 DHT.Node
	n2.Create("127.0.0.1:2001")
	n2.Run()
	n2.Join("127.0.0.1:2000")

	time.Sleep(3 * time.Second)
	n1.Dump()
	n2.Dump()

	n2.Put("ccc", "ddd")

	fmt.Println(n1.Get("ccc"))
}
