package main

import (
	"../src"
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"time"
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

func main() {

	//s,_:=syscall.Socket(syscall.AF_INET,syscall.SOCK_STREAM,syscall.IPPROTO_TCP)
	//var reuse byte=1
	//syscall.Setsockopt(s,syscall.SOL_SOCKET,syscall.SO_REUSEADDR,&reuse,int32(unsafe.Sizeof(reuse)))
	init1()
	var nodes [160]DHT.Node
	localAddress := DHT.GetLocalAddress()
	fmt.Println("local address: " + localAddress)

	port := 3000
	nodes[0].Create(localAddress + ":3000")
	nodes[0].Run()

	kvMap := make(map[string]string)
	var nodecnt = 1
	for i := 0; i < 5; i++ {
		//join 30 nodes
		for j := 0; j < 30; j++ {
			var index = i*30 + j + 1
			port++
			nodes[index].Create(localAddress + ":" + strconv.Itoa(port))
			nodes[index].Run()
			if !nodes[index].Join(localAddress + ":" + strconv.Itoa(3000+5*i)) {
				log.Fatal("join failed")
			}
			time.Sleep(1 * time.Second)
			fmt.Println("port ", port, " joined at 3000")
		}
		nodecnt += 30
		time.Sleep(10 * time.Second)
		for j := i * 5; j <= i*30+30; j++ {
			nodes[j].Dump()
		}
		//put 300 kv
		for j := 0; j < 300; j++ {
			k := RandStringRunes(30)
			v := RandStringRunes(30)
			kvMap[k] = v
			nodes[rand.Intn(nodecnt)+i*5].Put(k, v)
		}
		//get 300 kv and check correctness
		var keyList [500]string
		cnt := 0
		for k, v := range kvMap {
			if cnt == 200 {
				break
			}
			var tmp = rand.Intn(nodecnt) + i*5
			success, fetchedVal := nodes[tmp].Get(k)
			if !success {
				success, fetchedVal = nodes[tmp].Get(k)
				log.Fatal("error:can't find key ", k, " from node ", tmp)
			}
			if fetchedVal != v {
				log.Fatal("actual: ", fetchedVal, " expected: ", v)
			}
			keyList[cnt] = k
			cnt++
		}
		//delete 300 kv
		for j := 0; j < 300; j++ {
			delete(kvMap, keyList[j])
			nodes[rand.Intn(nodecnt)+i*5].Del(keyList[j])
		}
		for j := i * 5; j <= i*30+30; j++ {
			nodes[j].Dump()
		}

		//force quit and join 5 nodes
		//for j := 0; j < 5; j++ {
		//	nodes[j+i*5+5].ForceQuit()
		//	time.Sleep(3 * time.Second)
		//	fmt.Println("force quit node ", j+i*5+5)
		//}
		//time.Sleep(10 * time.Second)
		//for j := 0; j < 5; j++ {
		//	nodes[j+i*5+5].Create(localAddress + ":" + strconv.Itoa(j + i*5 + 5 + 3000))
		//	nodes[j+i*5+5].Run()
		//	if !nodes[j+i*5+5].Join(localAddress + ":" + strconv.Itoa(3000+i*5)) {
		//		log.Fatal("join failed")
		//	}
		//	fmt.Println("port ", j+i*5+5, " joined at ", 3000+i*5)
		//	time.Sleep(3 * time.Second)
		//}

		//quit 5 nodes
		for j := 0; j < 5; j++ {
			nodes[j+i*5].Quit()
			fmt.Println("quit ", j+i*5, " node")
			time.Sleep(3 * time.Second)
			nodes[j+i*5+1].Dump()
		}
		nodecnt -= 5
		time.Sleep(4 * time.Second)
		for j := i*5 + 5; j <= i*30+30; j++ {
			nodes[j].Dump()
		}
		//put 150 kv
		for j := 0; j < 150; j++ {
			k := RandStringRunes(30)
			v := RandStringRunes(30)
			kvMap[k] = v
			nodes[rand.Intn(nodecnt)+i*5+5].Put(k, v)
		}
		//get 200 kv and check correctness

		cnt = 0
		for k, v := range kvMap {
			if cnt == 200 {
				break
			}
			var tmp = rand.Intn(nodecnt) + i*5 + 5
			success, fetchedVal := nodes[tmp].Get(k)
			if !success {
				success, fetchedVal = nodes[tmp].Get(k)
				log.Fatal("error:can't find key ", k, " from node ", tmp)

			}
			if fetchedVal != v {
				log.Fatal("actual: ", fetchedVal, " expected: ", v)
			}
			keyList[cnt] = k
			cnt++
		}
		//delete 150 kv
		for j := 0; j < 150; j++ {
			delete(kvMap, keyList[j])
			nodes[rand.Intn(nodecnt)+i*5+5].Del(keyList[j])
		}
		time.Sleep(4 * time.Second)
	}
}
