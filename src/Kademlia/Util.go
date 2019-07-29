package Kademlia

import (
	"math/big"
	"sort"
)

func (this *Node) getDis(contact Contact) *big.Int {
	var z big.Int
	z.Xor(this.self.NodeNum, contact.NodeNum)
	return &z
}

func (this *Node) CalcPrefix(num *big.Int) int {
	var Mod = new(big.Int).Exp(big.NewInt(2), big.NewInt(M), nil)
	for i := M - 1; i >= 1; i-- {
		Mod.Div(Mod, big.NewInt(2))
		var a, b big.Int
		a.Div(this.self.NodeNum, Mod)
		b.Div(num, Mod)
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
	if len(contacts) >= num {
		return contacts[0:num]
	} else {
		return contacts
	}
}
