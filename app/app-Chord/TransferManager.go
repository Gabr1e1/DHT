package app

import (
	"errors"
	"fmt"
	"net/rpc"
	"time"
)

func (this *Peer) checkChoke(addr string) bool {
	return false
}

const maxTry = 3

func (this *Peer) Connect(other PeerInfo) (*rpc.Client, error) {
	c := make(chan *rpc.Client, 1)
	var err error
	var client *rpc.Client

	go func() {
		for i := 0; i < maxTry; i++ {
			client, err = rpc.Dial("tcp", other.addr)
			if err == nil {
				c <- client
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
	}()

	select {
	case client := <-c:
		//fmt.Println("Call Successful")
		return client, nil
	case <-time.After(666 * time.Millisecond):
		fmt.Println("Can't Connect ", other)
		if err == nil {
			err = errors.New("can't connect")
		}
		return nil, err
	}
}

type Request struct {
	Addr     string
	PieceNum int
}

func (this *Peer) UploadData(req *Request, reply *[]byte) error {
	choke := this.checkChoke(req.Addr)
	if choke {
		return errors.New("NotGivingYouDataBitch")
	}
	//fmt.Println(req.Addr, "trying to read", this.Self.addr, req.PieceNum)
	*reply = this.readFile(req.PieceNum)
	return nil
}
