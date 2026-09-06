package service

import (
	"testing"
	"time"
)

func TestRankTop10(t *testing.T) {
	r := Rank(MockUniverse(), time.Now(), 10000, 0)
	if r.Count == 0 || r.Count > 10 {
		t.Fatalf("count = %d", r.Count)
	}
	for i := 1; i < len(r.Stocks); i++ {
		if r.Stocks[i].GLScore > r.Stocks[i-1].GLScore {
			t.Fatal("tidak descending")
		}
		if r.Stocks[i].Rank != i+1 {
			t.Fatal("rank salah")
		}
	}
}

func TestRankEmpty(t *testing.T) {
	r := Rank(nil, time.Now(), 10000, 0)
	if len(r.Stocks) != 0 {
		t.Fatal("universe kosong harus []")
	}
}
