package Kademlia

import (
	"math/rand"
)

const K = 20

type KBucket struct {
	contacts []Contact
}

func (this *KBucket) size() int {
	return len(this.contacts)
}

func (this *KBucket) insert(node *Node, contact Contact) {
	for i := 0; i < len(this.contacts); i++ {
		if contact.IPAddr == this.contacts[i].IPAddr {
			t := this.contacts[i]
			this.contacts = append(this.contacts[0:i], this.contacts[i+1:]...)
			this.contacts = append(this.contacts, t)
			return
		}
	}

	if len(this.contacts) < K {
		this.contacts = append(this.contacts, contact)
		return
	}
	if !node.Ping(contact) {
		this.contacts = append(this.contacts[1:], contact)
	}
}

func (this *KBucket) Get(K int) []Contact {
	rand.Shuffle(len(this.contacts), func(i, j int) {
		this.contacts[i], this.contacts[j] = this.contacts[j], this.contacts[i]
	})
	if len(this.contacts) < K {
		return this.contacts
	} else {
		return this.contacts[0:K]
	}
}
