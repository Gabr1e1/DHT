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

	init1()
	var nodes [200]DHT.Node
	localAddress := DHT.GetLocalAddress()
	fmt.Println(localAddress)

	port := 1000
	nodes[0].Create(localAddress + ":" + strconv.Itoa(port))
	nodes[0].Run()

	kvMap := make(map[string]string)
	var nodecnt = 1

	for i := 0; i < 5; i++ {
		//join 15 nodes
		for j := 0; j < 30; j++ {
			var index = i*30 + j + 1
			port++
			nodes[index].Create(localAddress + ":" + strconv.Itoa(port))
			nodes[index].Run()
			nodes[index].Join(localAddress + ":" + strconv.Itoa(1000))
			time.Sleep(1 * time.Second)
			fmt.Println("port ", port, " joined at 1000")
		}
		nodecnt += 30
		time.Sleep(30 * time.Second)

		//put 300 kv
		for j := 0; j < 300; j++ {
			k := RandStringRunes(30)
			v := RandStringRunes(30)
			kvMap[k] = v
			tmp := rand.Intn(nodecnt)
			fmt.Println(j, tmp)
			nodes[tmp].Put(k, v)
		}

		//get 200 kv and check correctness
		var keyList [200]string
		cnt := 0
		for k, v := range kvMap {
			fmt.Println("Have done: ", cnt)
			if cnt == 200 {
				break
			}
			var tmp = rand.Intn(nodecnt)
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
			nodes[rand.Intn(nodecnt)].Del(keyList[j])
		}

		////force quit and join 5 nodes
		//for j := 0; j < 5; j++ {
		//	nodes[j+i*5+5].ForceQuit()
		//	time.Sleep(3 * time.Second)
		//	fmt.Println("force quit node ", j+i*5+5)
		//}
		//time.Sleep(10 * time.Second)
		//for j := i * 5; j <= i*15+15; j++ {
		//	if nodes[j].Node_.Listening {
		//		nodes[j].Dump()
		//	}
		//}
		//for j := 0; j < 5; j++ {
		//	nodes[j+i*5+5] = DHT.NewNode(j + i*5 + 5 + 1000)
		//	nodes[j+i*5+5].Run(&wg)
		//	nodes[j+i*5+5].Join(localAddress + ":" + strconv.Itoa(1000+i*5))
		//	fmt.Println("port ", j+i*5+5, " joined at ", 1000+i*5)
		//	time.Sleep(3 * time.Second)
		//}
		////quit 5 nodes
		//for j := 0; j < 5; j++ {
		//	nodes[j+i*5].Quit()
		//	fmt.Println("quit ", j+i*5, " node")
		//	time.Sleep(3 * time.Second)
		//	nodes[j+i*5+1].Dump()
		//}
		//nodecnt -= 5
		//time.Sleep(4 * time.Second)
		//for j := i*5 + 5; j <= i*15+15; j++ {
		//	nodes[j].Dump()
		//}
		////put 300 kv
		//for j := 0; j < 300; j++ {
		//	k := RandStringRunes(30)
		//	v := RandStringRunes(30)
		//	kvMap[k] = v
		//	nodes[rand.Intn(nodecnt)+i*5+5].Put(k, v)
		//}
		////get 200 kv and check correctness
		//
		//cnt = 0
		//for k, v := range kvMap {
		//	if cnt == 200 {
		//		break
		//	}
		//	var tmp = rand.Intn(nodecnt) + i*5 + 5
		//	fetchedVal, success := nodes[tmp].Get(k)
		//	if !success {
		//		fetchedVal, success = nodes[tmp].Get(k)
		//		log.Fatal("error:can't find key ", k, " from node ", tmp)
		//
		//	}
		//	if fetchedVal != v {
		//		log.Fatal("actual: ", fetchedVal, " expected: ", v)
		//	}
		//	keyList[cnt] = k
		//	cnt++
		//}
		////delete 150 kv
		//for j := 0; j < 150; j++ {
		//	delete(kvMap, keyList[j])
		//	nodes[rand.Intn(nodecnt)+i*5+5].Del(keyList[j])
		//}
	}
}
