package DHT

import (
	"errors"
	"fmt"
	"net"
	"net/rpc"
	"time"
)

const maxTry = 3

func (n *Node) Connect(otherNode InfoType) (*rpc.Client, error) {
	//fmt.Println("Calling: ", otherNode)
	if otherNode.IPAddr == "" {
		return nil, errors.New("invalid address")
	}

	c := make(chan *rpc.Client, 1)
	var err error
	var client *rpc.Client

	go func() {
		for i := 0; i < maxTry; i++ {
			client, err = rpc.Dial("tcp", otherNode.IPAddr)
			if err == nil {
				c <- client
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
	}()

	select {
	case client := <-c:
		//fmt.Println("Call Successful")
		return client, nil
	case <-time.After(666 * time.Millisecond):
		fmt.Println("Can't Connect ", otherNode)
		if err == nil {
			err = errors.New("can't connect")
		}
		return nil, err
	}
}

func GetLocalAddress() string {
	var localaddress string

	ifaces, err := net.Interfaces()
	if err != nil {
		panic("init: failed to find network interfaces")
	}

	// find the first non-loopback interface with an IP address
	for _, elt := range ifaces {
		if elt.Flags&net.FlagLoopback == 0 && elt.Flags&net.FlagUp != 0 {
			addrs, err := elt.Addrs()
			if err != nil {
				panic("init: failed to get addresses for network interface")
			}

			for _, addr := range addrs {
				if ipnet, ok := addr.(*net.IPNet); ok {
					if ip4 := ipnet.IP.To4(); len(ip4) == net.IPv4len {
						localaddress = ip4.String()
						break
					}
				}
			}
		}
	}
	if localaddress == "" {
		panic("init: failed to find non-loopback interface with valid address on this node")
	}

	return localaddress
}
