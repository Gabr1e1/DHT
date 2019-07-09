package main

import (
	"crypto/sha1"
)

func getHash(k string) int {
	h := sha1.New()
	h.Write([]byte(k))
	hRes := h.Sum(nil)[0:4]
	var hash int = 0
	for i := range hRes {
		hash = hash*256 + int(hRes[i])
	}
	return hash
}
