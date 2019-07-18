package main

import (
	"fmt"
	"math/big"
	"sync"
	"time"
)

var wg sync.WaitGroup

func print() {
	defer wg.Done()
	for {
		fmt.Println("123")
		time.Sleep(time.Millisecond * 100)
	}
}

func fun() {
	//fmt.Println(DHT.GetHash("fpMloialW9VECECBwqv1zC5j78diWv"))
	wg.Add(1)
	go print()
}

func main() {
	var a = big.NewInt(1)
	var b = a
	a.Add(a, a)
	fmt.Println(b.String())
}
