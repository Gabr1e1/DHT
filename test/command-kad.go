package main

import (
	"../src/Kademlia"
	"bufio"
	"fmt"
	"os"
	"strconv"
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
			port, _ := strconv.Atoi(words[1])
			cur.Run(port)
		case "join":
			cur.Join(words[1])
		case "put":
			cur.Put(words[1], words[2])
		case "get":
			ok, val := cur.Get(words[1])
			fmt.Println(ok, val)
		case "ping":
			ok := cur.Ping(Kademlia.Contact{Ip: words[1]})
			fmt.Println(ok)
		default:
			fmt.Println("FUCKING WRONG COMMAND")
		}
	}
}
