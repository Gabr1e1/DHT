package main

import (
	"../src/Kademlia"
	"bufio"
	"fmt"
	"os"
	"strings"
)

func main() {
	reader := bufio.NewReader(os.Stdin)
	var cur *Kademlia.Node

	for {
		cmd, _, _ := reader.ReadLine()
		words := strings.Fields(string(cmd))

		switch string(words[0]) {
		case "create":
			cur = new(Kademlia.Node)
			cur.Create(words[1])
		case "run":
			cur.Run()
		case "join":
			cur.Join(words[1])
		case "put":
			cur.Put(words[1], words[2])
		case "get":
			ok, val := cur.Get(words[1])
			fmt.Println(ok, val)
		default:
			fmt.Println("FUCKING WRONG COMMAND")
		}
	}
}
