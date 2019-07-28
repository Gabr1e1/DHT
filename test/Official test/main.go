package main

import (
	"../../src/Chord"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"time"
)

var (
	help bool

	version bool

	level int

	nointerrupt bool
)

func init() {
	flag.BoolVar(&help, "h", false, "give help")
	flag.BoolVar(&version, "v", false, "version")
	flag.BoolVar(&nointerrupt, "n", false, "can not use ctrl-c to interrupt program")
	flag.IntVar(&level, "l", 0, "levels of tests(0 for standard and advanced, 1 for advanced, 2 for addition, 3 for all)")

	flag.Usage = usage

	flag.Parse()

	fmt.Println("ppca-dht 2019-7 v0.0.1")

	if help {
		flag.Usage()
		os.Exit(0)
	}
	if version {
		os.Exit(0)
	}

	if nointerrupt {
		go func() {
			for {
				c := make(chan os.Signal, 1)
				signal.Notify(c, os.Interrupt, os.Kill)

				s := <-c
				fmt.Println("Got signal:", s)
			}
		}()
	}
}

func failrate() float64 {
	return float64(totalFail) / float64(totalCnt)
}

func main() {
	file, _ := os.Create("error.log")
	DHT.RedirectStderr(file)

	green.Println("Start Testing")
	switch level {
	case -1:
		naiveTest()
	case 3:
		blue.Println("Start Additive Tests")
		if maxFail > failrate() {
			green.Println("Passed with", failrate())
		} else {
			red.Println("Failed")
			os.Exit(0)
		}
		fallthrough
	case 0:
		blue.Println("Start Standard Tests")

		if standardTest(); maxFail > failrate() {
			green.Println("Passed Standard Tests with", failrate())
		} else {
			red.Println("Failed Standard Tests")
			os.Exit(0)
		}
		fallthrough
	case 1:
		blue.Println("Start Advanced Tests")
		if advancedTest(); maxFail > failrate() {
			green.Println("Passed Advanced Tests with", failrate())
		} else {
			red.Println("Failed Advanced Tests")
			os.Exit(0)
		}
	case 2:
		blue.Println("Start Additive Tests")
		if maxFail > failrate() {
			green.Println("Passed with", failrate())
		} else {
			red.Println("Failed")
			os.Exit(0)
		}
	default:
		red.Print("Select error, ask -h for help")
		os.Exit(0)
	}
	fmt.Println(time.Now())
}

func usage() {
	fmt.Fprintf(os.Stderr, `Usage: ppca-dht [-h help] [-v version] [-a addition-test]
Options:
`)
	flag.PrintDefaults()
}
