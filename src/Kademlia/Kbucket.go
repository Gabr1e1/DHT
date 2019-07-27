package Kademlia

const K = 20

type Kbucket struct {
	contacts []Contact
}

func (this *Kbucket) insert(node *Node, contact Contact) {
	if len(this.contacts) < K {
		this.contacts = append(this.contacts, contact)
		return
	}
	if !node.Ping(contact) {
		this.contacts = append(this.contacts[1:], contact)
	}
}

func (this *Kbucket) Get(K int)
