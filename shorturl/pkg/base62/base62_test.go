package base62

import (
	"testing"
)

func TestInt2String(t *testing.T) {
	tests := []struct {
		name string
		seq  uint64
		want string
	}{
		// TODO: Add test cases.
		{name: "case: 0", seq: 0, want: "0"},
		{name: "case: 1", seq: 1, want: "1"},
		{name: "case: 2", seq: 62, want: "10"},
		{name: "case: 3", seq: 6347, want: "1En"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Int2String(tt.seq); got != tt.want {
				t.Errorf("Int2String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestString2Int(t *testing.T) {
	tests := []struct {
		name    string
		s       string
		wantSeq uint64
	}{
		// TODO: Add test cases.
		{name: "case: 1", s: "0", wantSeq: 0},
		{name: "case: 2", s: "1", wantSeq: 1},
		{name: "case: 3", s: "10", wantSeq: 62},
		{name: "case: 4", s: "1En", wantSeq: 6347},
		{name: "case: 5", s: "1En", wantSeq: 6347},
		{name: "case: 6", s: "1En", wantSeq: 6347},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotSeq := String2Int(tt.s); gotSeq != tt.wantSeq {
				t.Errorf("String2Int() = %v, want %v", gotSeq, tt.wantSeq)
			}
		})
	}
}
