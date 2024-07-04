package base62

import (
	"math"
	"slices"
	"strings"
)

var (
	base62str string
)

func MustInit(bs string) {
	if len(bs) != 62 {
		panic("need base string 62 bit")
	}
	base62str = bs
}

/* 把uint64的数字转换位62进制的string */
func Int2String(seq uint64) string {
	ret := make([]byte, 0)
	if seq == 0 {
		return "0"
	}
	for seq > 0 {
		ret = append(ret, base62str[seq%62])
		seq = seq / 62
	}
	slices.Reverse(ret)
	return string(ret)
}

/* String2Int 把62进制的string转换为uint64的数字 */
func String2Int(s string) (seq uint64) {
	bs := []byte(s)
	slices.Reverse(bs)
	for idx, b := range bs {
		base := math.Pow(62, float64(idx))
		seq += uint64(strings.Index(base62str, string(b))) * uint64(base)
	}
	return seq
}
