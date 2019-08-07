package main

import (
	"../src/Chord"
	"../src/Kademlia"
	"fmt"
	"log"
	"math/rand"
	_ "net/http/pprof"
	"strconv"
)

func init1() {
	rand.Seed(1)
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

func RandStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

type keyval struct {
	key string
	val string
}

const SIZE = 100

func main() {
	//create & join
	var node [SIZE]Kademlia.Node
	for i := 0; i < SIZE; i++ {
		node[i].Create(DHT.GetLocalAddress() + ":" + strconv.Itoa(i+2000))
		node[i].Run(i + 2000)
	}

	for i := 1; i < SIZE; i++ {
		fmt.Println("Joining", i)
		node[i].Join(node[0].Self.Ip)
	}
	for i := 0; i < SIZE; i++ {
		node[i].Dump()
	}

	//put
	for i := 0; i < 100; i++ {
		fmt.Println("Putting", i)
		node[rand.Intn(SIZE)].Put(strconv.Itoa(i), strconv.Itoa(i))
		node[rand.Intn(SIZE)].Put(strconv.Itoa(i), strconv.Itoa(i+1))
	}
	for i := 0; i < SIZE; i++ {
		node[i].Dump()
	}

	//get
	for i := 0; i < 100; i++ {
		fmt.Println("Getting", i)
		ok, val := node[rand.Intn(SIZE)].Get(strconv.Itoa(i))
		if !ok {
			log.Fatal("FUCK, can't get")
		}
		fmt.Println(val)
	}
}
