package app

import (
	"encoding/binary"
	"strconv"
	"strings"
)

func parsePeer(value string) []PeerInfo {
	var ret []PeerInfo
	b := []byte(value)
	for i := 0; i < len(b); i += 6 {
		var cur string
		for j := 0; j < 4; j++ {
			cur += strconv.Itoa(int(b[j])) + "."
		}
		cur = cur[:len(cur)-1] + ":"
		cur += strconv.Itoa(int(int(b[i+4])*256 + int(b[i+5])))
		ret = append(ret, PeerInfo{cur})
	}
	return ret
}

func encodePeer(addr string) string {
	var ret []byte
	for strings.Index(addr, ".") > -1 || strings.Index(addr, ":") > -1 {
		t := strings.Index(addr, ".")
		if t == -1 {
			t = strings.Index(addr, ":")
		}
		curNum, _ := strconv.Atoi(addr[0:t])
		ret = append(ret, byte(curNum))
		addr = addr[t+1:]
	}
	curNum, _ := strconv.Atoi(addr)

	port := make([]byte, 2)
	binary.BigEndian.PutUint16(port, uint16(curNum))
	ret = append(ret, port...)
	return string(ret)
}

func parseMagnetLink(link string) (bool, string, string) {
	i := strings.Index(link, "?")
	if i > -1 {
		if link[0:i] != "magnet:" {
			return false, "", ""
		}
		if link[i+1:i+6] != "hash=" {
			return false, "", ""
		}
		j := strings.Index(link[i+1:], "?") + i + 1
		if j > -1 {
			if link[j+1:j+4] != "fn=" {
				return false, "", ""
			}
			return true, link[i+6 : j], link[j+4:]
		}
	}
	return false, "", ""
}
