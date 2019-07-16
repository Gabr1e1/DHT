package DHT

import (
	"crypto/sha1"
)

func GetHash(k string) int {
	h := sha1.New()
	h.Write([]byte(k))
	hRes := h.Sum(nil)[0:4]
	var hash int = 0
	for i := range hRes {
		hash = hash*256 + int(hRes[i])
	}
	return hash % 4294967296
}
