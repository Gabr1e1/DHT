package Kademlia

import (
	"fmt"
	"math/big"
	"sort"
)

func (this *Node) getDis(contact Contact) *big.Int {
	var z big.Int
	z.Xor(this.Self.NodeNum, contact.NodeNum)
	return &z
}

func (this *Node) CalcPrefix(num *big.Int) int {
	//fmt.Println(this.Self.NodeNum.String(), num.String())
	if this.Self.NodeNum.Cmp(num) == 0 {
		return M
	}
	var Div = big.NewInt(1)
	for i := M - 1; i >= 1; i-- {
		Div.Mul(Div, big.NewInt(2))
		var a, b big.Int
		a.Div(this.Self.NodeNum, Div)
		b.Div(num, Div)
		if a.Cmp(&b) == 0 {
			return i
		}
	}
	return 0
}

func verifyIdentity(A Contact, B Contact) bool {
	return A.NodeNum.Cmp(B.NodeNum) == 0 && A.IPAddr == B.IPAddr
}

func (this *Node) GetClosestInList(num int, contacts []Contact) []Contact {
	sort.Slice(contacts, func(i, j int) bool {
		return this.getDis(contacts[i]).Cmp(this.getDis(contacts[j])) < 0
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
