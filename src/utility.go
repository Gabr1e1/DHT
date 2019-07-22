package DHT

import "math/big"

func copyInfo(t InfoType) InfoType {
	return InfoType{t.IPAddr, new(big.Int).Set(t.NodeNum)}
}