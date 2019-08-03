package torrent_Kad

import (
	"strings"
)

func (this *BEncoding) peekNext() byte {
	ret, _ := this.reader.ReadByte()
	_ = this.reader.UnreadByte()
	return ret
}

func (this *BEncoding) readNext() byte {
	ret, _ := this.reader.ReadByte()
	return ret
}

func (this *BEncoding) ReadAnything() interface{} {
	t := this.peekNext()
	switch t {
	case 'i':
		this.readNext()
		return this.ReadInt()
	case 'l':
		this.readNext()
		return this.ReadList()
	case 'd':
		this.readNext()
		return this.ReadDict()
	default:
		return this.ReadString()
	}
}

func (this *BEncoding) ReadString() string {
	num := 0
	for this.peekNext() != ':' {
		cur := this.readNext()
		num = num*10 + int(cur) - int('0')
	}
	this.readNext()
	ret := ""
	for i := 0; i < num; i++ {
		ret += string(this.readNext())
	}
	return ret
}

func (this *BEncoding) ReadInt() int {
	ret := 0
	sgn := 1
	for this.peekNext() != 'e' {
		cur := this.readNext()
		if cur == '-' {
			sgn = -1
		} else {
			ret = ret*10 + int(cur) - int('0')
		}
	}
	this.readNext() //read the closing 'e'
	return sgn * ret
}

func (this *BEncoding) ReadList() []interface{} {
	var ret []interface{}
	for this.peekNext() != 'e' {
		ret = append(ret, this.ReadAnything())
	}
	this.readNext()
	return ret
}

func (this *BEncoding) ReadDict() map[interface{}]interface{} {
	ret := make(map[interface{}]interface{})
	var key, value interface{}
	key = nil
	for this.peekNext() != 'e' {
		if key != nil {
			value = this.ReadAnything()
			ret[key] = value
			key = nil
		} else {
			key = this.ReadAnything()
		}
	}
	this.readNext()
	return ret
}

func Parse(str string) interface{} {
	e := BEncoding{}
	e.reader = strings.NewReader(str)
	return e.ReadAnything()
}
