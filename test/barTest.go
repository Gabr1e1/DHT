package main

import (
	"fmt"
	"time"

	"github.com/cheggaaa/pb/v3"
)

func main() {
	count := 1000
	// create and start new bar
	bar := pb.StartNew(count)

	// start bar from 'default' template
	// bar := pb.Default.Start(count)

	// start bar from 'simple' template
	// bar := pb.Simple.Start(count)

	// start bar from 'full' template
	// bar := pb.Full.Start(count)

	for i := 0; i < count; i++ {
		bar.Increment()
		time.Sleep(time.Millisecond)
	}
	bar.Finish()
	fmt.Println("1234")
	time.Sleep(100 * time.Second)
}