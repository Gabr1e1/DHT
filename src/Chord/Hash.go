package DHT

import (
	"crypto/sha1"
	"fmt"
	"math/big"
)

func GetHash(k string) *big.Int {
	h := sha1.New()
	h.Write([]byte(k))
	hRes := h.Sum(nil)
	var hash big.Int
	hash.SetBytes(hRes)
	return &hash
}

func Hex2Num(b byte) byte {
	if b >= byte('0') && b <= byte('9') {
		return b - byte('0')
	}
	return b - byte('A') + 10
}

func GetByteHash(k string) string {
	t := fmt.Sprintf("%x", GetHash(k))
	var ret []byte
	for i := 0; i < len(t)/2; i++ {
		ret = append(ret, Hex2Num(t[i*2])*16+Hex2Num(t[i*2+1]))
	}
	return string(ret)
}
