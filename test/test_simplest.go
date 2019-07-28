package main

import (
	"../src/Chord"
	"fmt"
	"time"
)

func main() {
	var n1 DHT.Node
	n1.Create_("127.0.0.1:2000")
	n1.Create()
	n1.Run()
	n1.Put("aaa", "bbb")

	var n2 DHT.Node
	n2.Create_("127.0.0.1:2001")
	n2.Create()
	n2.Run()
	n2.Join("127.0.0.1:2000")

	var n3 DHT.Node
	n3.Create_("127.0.0.1:2002")
	n3.Create()
	n3.Run()
	n3.Join("127.0.0.1:2000")

	time.Sleep(10 * time.Second)

	n1.Dump()
	n2.Dump()
	n3.Dump()

	//Add
	for i := 'a'; i <= 'z'; i++ {
		//fmt.Println("HASH: ", DHT.GetHash(string(i)))
		n1.Put(string(i), string(i))
	}

	for i := 'a'; i <= 'z'; i++ {
		ok, str := n2.Get(string(i))
		fmt.Println(i, ok, str)
	}

	for i := 'A'; i <= 'Z'; i++ {
		//fmt.Println(DHT.GetHash(string(i)))
		n2.Put(string(i), string(i))
	}
	for i := 'A'; i <= 'Z'; i++ {
		ok, str := n1.Get(string(i))
		fmt.Println(i, ok, str)
	}

	n1.Dump()
	n2.Dump()
	n3.Dump()

	fmt.Println("Start Force Quit")
	n1.Quit()
	n2.Quit()

	time.Sleep(10 * time.Second)

	for i := 'A'; i <= 'Z'; i++ {
		ok, str := n3.Get(string(i))
		fmt.Println(i, ok, str)
	}
}
