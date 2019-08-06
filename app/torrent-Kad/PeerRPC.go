package torrent_Kad

import (
	"errors"
	"fmt"
	"net/rpc"
	"strconv"
	"time"
)

type TorrentRequest struct {
	Infohash string
	Index    int
	Length   int
}

type IntSet map[int]struct{}

const maxTry = 3

func (this *Peer) Connect(addr string) (*rpc.Client, error) {
	c := make(chan *rpc.Client, 1)
	var err error
	var client *rpc.Client

	go func() {
		for i := 0; i < maxTry; i++ {
			client, err = rpc.Dial("tcp", addr)
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
		fmt.Println("Can't Connect ", addr)
		if err == nil {
			err = errors.New("can't connect")
		}
		return nil, err
	}
}

func (this *Peer) GetTorrentFile(hashId *string, file *[]byte) error {
	if info, ok := this.FileStat[*hashId]; ok {
		*file = info.Torrent
		return nil
	}
	return errors.New("Can't find torrent corresponding to " + *hashId)
}

func (this *Peer) GetPieceStatus(hashId *string, stat *IntSet) error {
	if info, ok := this.FileStat[*hashId]; ok {
		*stat = info.Pieces
		return nil
	}
	return errors.New("Can't find piece corresponding to " + *hashId)
}

func (this *Peer) GetPiece(request *TorrentRequest, pieces *[]byte) error {
	if info, ok := this.FileStat[request.Infohash]; ok {
		fmt.Println("Transferring piece", request.Index)
		*pieces = info.GetFileInfo(request.Index, request.Length)
		return nil
	}
	return errors.New("Can't get piece" + strconv.Itoa(request.Index))
}
