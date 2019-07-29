package main

import (
	"../src"
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"time"
)

const size = 15

func main() {
	var node [100]DHT.Node

	for i := 0; i < size; i++ {
		fmt.Println(i)
		node[i].Create("127.0.0.1:" + strconv.Itoa(i+16000))
		node[i].Run()
	}
	for i := 1; i < size; i++ {
		fmt.Println(i)
		node[i].Join("127.0.0.1:16000")
		time.Sleep(1 * time.Second)
	}
	time.Sleep(30 * time.Second)
	for i := 0; i < size; i++ {
		fmt.Println(node[i].Finger[0].NodeNum)
	}

	//Put & GetClosest
	for i := 1; i <= 1000; i++ {
		fmt.Println(i)
		node[rand.Intn(size)].Put(strconv.Itoa(i), strconv.Itoa(i))
		if i%300 == 0 {
			time.Sleep(5 * time.Second)
		}
	}
	for i := 1; i <= 1000; i++ {
		ok, val := node[rand.Intn(size)].Get(strconv.Itoa(i))
		fmt.Println(i, ok, val)
		if i%300 == 0 {
			time.Sleep(5 * time.Second)
		}
		if ok != true || val != strconv.Itoa(i) {
			log.Fatal("WRONG!")
		}
	}

	//Del & GetClosest again
	for i := 1; i <= 1000; i++ {
		fmt.Println(i)
		node[rand.Intn(size)].Del(strconv.Itoa(i))
		if i%300 == 0 {
			time.Sleep(5 * time.Second)
		}
	}

	for i := 1; i <= 1000; i++ {
		ok, val := node[rand.Intn(size)].Get(strconv.Itoa(i))
		fmt.Println(i, ok, val)
		if i%300 == 0 {
			time.Sleep(5 * time.Second)
		}
		if ok != false {
			log.Fatal("WRONG!")
		}
	}
}
