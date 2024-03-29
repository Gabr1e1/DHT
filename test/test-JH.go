package main

import (
	"../src"
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"time"
)

const second = 1000 * time.Millisecond

var MAP map[string]string
var id int
var node [20000]DHT.Node
var PUT int

func KVTest() {
	fmt.Println("Sleep 30 seconds")
	time.Sleep(5 * second)

	// insert
	fmt.Println("Start to test insert")
	for i := 0; i < 700; i++ {
		str := strconv.Itoa(PUT)
		//k, v := randString(10), randString(10)
		//MAP[k] = v
		MAP[str] = str
		p := rand.Int() % id
		//(*node[p]).Put(k, v)
		node[p].Put(str, str)
		PUT++
	}

	// check correctness
	fmt.Println("Start to check correctness")
	cnt := 0
	for k, v := range MAP {
		p := rand.Int() % id
		_, res := node[p].Get(k)
		if res != v {
			log.Fatalln("GetClosest incorrect when get key", k)
		}
		cnt++
		if cnt == 400 {
			break
		}
	}

	// delete
	fmt.Println("Start to test delete")
	cnt = 0
	var str [300]string
	for k := range MAP {
		str[cnt] = k
		cnt++
		if cnt == 300 {
			break
		}
	}
	for _, k := range str {
		node[rand.Int()%id].Del(k)
		delete(MAP, k)
	}

	fmt.Println("Sleep 10 seconds")
	time.Sleep(5 * second)
}

func main() {
	fmt.Println("Start time: ", time.Now())
	//randomInit()
	rand.Seed(1)
	MAP = make(map[string]string)

	id = 0

	node[id].Create(DHT.GetLocalAddress() + ":" + strconv.Itoa(2000))
	node[id].Run()
	id++

	localAddr := DHT.GetLocalAddress()

	for t := 0; t < 5; t++ {
		fmt.Println("Start to test join, Round", t)
		for i := 0; i < 30; i++ {
			node[id].Create(localAddr + ":" + strconv.Itoa(id+2000))
			node[id].Run()
			node[id].Join(localAddr + ":" + strconv.Itoa(2000+rand.Int()%id))
			id++

			time.Sleep(1 * second)
		}

		fmt.Println("Sleep 5 seconds")
		time.Sleep(5 * second)
		for i := 0; i < id; i++ {
			node[i].Dump()
		}

		KVTest()

		fmt.Println("Start to test quit")
		for i := 10; i >= 1; i-- {
			node[id-i].Quit()
			time.Sleep(3 * second)
		}
		id -= 10

		fmt.Println("Sleep 5 seconds")
		time.Sleep(5 * second)
		for i := 0; i < id; i++ {
			node[i].Dump()
		}

		KVTest()
	}
	fmt.Println("End time: ", time.Now())

}
