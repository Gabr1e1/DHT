package Kademlia

import (
	"math/rand"
	"sync"
)

const K = 20

type KBucket struct {
	contacts []Contact
	mux      sync.Mutex
}

func (this *KBucket) size() int {
	return len(this.contacts)
}

func (this *KBucket) insert(node *Node, contact Contact) {
	this.mux.Lock()
	for i := 0; i < len(this.contacts); i++ {
		if contact.Ip == this.contacts[i].Ip {
			t := this.contacts[i]
			this.contacts = append(this.contacts[0:i], this.contacts[i+1:]...)
			this.contacts = append(this.contacts, t)
			this.mux.Unlock()
			return
		}
	}

	if len(this.contacts) < K {
		this.contacts = append(this.contacts, contact)
		this.mux.Unlock()
		return
	}
	if !node.Ping(this.contacts[0]) {
		this.contacts = append(this.contacts[1:], contact)
	}
	this.mux.Unlock()
}

func (this *KBucket) Get(K int) []Contact {
	this.mux.Lock()
	rand.Shuffle(len(this.contacts), func(i, j int) {
		this.contacts[i], this.contacts[j] = this.contacts[j], this.contacts[i]
	})
	if len(this.contacts) < K {
		this.mux.Unlock()
		return this.contacts
	} else {
		this.mux.Unlock()
		return this.contacts[0:K]
	}
}
