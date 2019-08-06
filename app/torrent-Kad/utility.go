package torrent_Kad

import "strings"

const maxTorrentSize = pieceSize * 80

func Min(x, y int) int {
	if x < y {
		return x
	} else {
		return y
	}
}

func parseMagnet(link string) (string, string, string) {
	l1 := strings.Index(link, "magnet:?xt=urn:btih:") + len("magnet:?xt=urn:btih:")
	l2 := strings.Index(link, "&dn=") + len("&dn=")
	l3 := strings.Index(link, "&tr=") + len("&tr=")
	if l1 < 0 || l2 < 0 || l3 < 0 {
		return "", "", ""
	}
	return link[l1 : l2-4], link[l2 : l3-4], link[l3:]
}
