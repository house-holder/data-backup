package main

import "testing"

func TestHashFunction1(t *testing.T) {
	tgt := "f1ebecbffecf3f8e4f60db92de00b600ee7b695c30f255463d55b36ba4ae35d6"
	got := getSHA256([]byte("some test string"))
	if got != tgt {
		t.Errorf("got %s, target %s", got, tgt)
	}
}

func TestHashFunction2(t *testing.T) {
	tgt := "9271675f13b85ffee2af5c98a4145382579ef20a2a5cb1310756357b5267090a"
	got := getSHA256([]byte("any test string"))
	if got != tgt {
		t.Errorf("got %s, target %s", got, tgt)
	}
}
