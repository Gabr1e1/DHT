package Kademlia

import (
	"fmt"
	"math/big"
	"sort"
)

func getDis(a *big.Int, b *big.Int) *big.Int {
	return new(big.Int).Xor(a, b)
}

func (this *Node) CalcPrefix(num *big.Int) int {
	//fmt.Println(this.Header.HashId.String(), num.String())
	if this.Self.Id.Cmp(num) == 0 {
		return M
	}
	var Div = big.NewInt(1)
	for i := M - 1; i >= 1; i-- {
		Div.Mul(Div, big.NewInt(2))
		var a, b big.Int
		a.Div(this.Self.Id, Div)
		b.Div(num, Div)
		if a.Cmp(&b) == 0 {
			return i
		}
	}
	return 0
}

func verifyIdentity(A Contact, B Contact) bool {
	return A.Id.Cmp(B.Id) == 0 && A.Ip == B.Ip
}

func GetClosestInList(id *big.Int, num int, contacts []Contact) []Contact {
	sort.Slice(contacts, func(i, j int) bool {
		return getDis(id, contacts[i].Id).Cmp(getDis(id, contacts[j].Id)) < 0
	})
	if len(contacts) > num {
		return contacts[0:num]
	} else {
		return contacts
	}
}

func (this *Node) Dump() {
	fmt.Println("Dumping Info About", this.Self)
	for i := 0; i < len(this.bucket); i++ {
		if len(this.bucket[i].contacts) == 0 {
			continue
		}
		fmt.Println("BUCKET", i, len(this.bucket[i].contacts), this.bucket[i].contacts)
	}
	fmt.Println("DATA: ")
	for _, v := range this.data {
		fmt.Print("{", v, "}")
	}
	fmt.Printf("\n\n")
}
