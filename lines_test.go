package main

import (
	"reflect"
	"testing"
)

func TestParseLines(t *testing.T) {
	texts, raws := parseLines([]byte("a\r\nb\nc"))
	if want := []string{"a", "b", "c"}; !reflect.DeepEqual(texts, want) {
		t.Fatalf("texts = %#v, want %#v", texts, want)
	}
	if want := []string{"a\r\n", "b\n", "c"}; !reflect.DeepEqual(raws, want) {
		t.Fatalf("raws = %#v, want %#v", raws, want)
	}
	if texts, raws := parseLines(nil); len(texts) != 0 || len(raws) != 0 {
		t.Fatalf("empty content produced %#v / %#v", texts, raws)
	}
}

func TestMarkKept(t *testing.T) {
	tests := []struct {
		name     string
		original []string
		edited   []string
		want     []bool
		ok       bool
	}{
		{"unchanged", []string{"a", "b", "c"}, []string{"a", "b", "c"}, []bool{true, true, true}, true},
		{"delete middle", []string{"a", "b", "c"}, []string{"a", "c"}, []bool{true, false, true}, true},
		{"delete all", []string{"a", "b"}, nil, []bool{false, false}, true},
		{"added line", []string{"a", "b"}, []string{"a", "x", "b"}, nil, false},
		{"modified line", []string{"a", "b"}, []string{"a", "B"}, nil, false},
		{"reordered", []string{"a", "b", "c"}, []string{"b", "a", "c"}, nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := markKept(tt.original, tt.edited)
			if ok != tt.ok {
				t.Fatalf("ok = %v, want %v", ok, tt.ok)
			}
			if ok && !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got %#v, want %#v", got, tt.want)
			}
		})
	}
}
