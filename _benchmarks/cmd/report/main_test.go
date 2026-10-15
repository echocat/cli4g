package main

import "testing"

func TestMedianPreservesSourceMetrics(t *testing.T) {
	points := []measurement{
		{nanoseconds: 7, bytes: 1, allocs: 6, characters: 123},
		{nanoseconds: 1, bytes: 7, allocs: 4, characters: 123},
		{nanoseconds: 6, bytes: 2, allocs: 3, characters: 123},
		{nanoseconds: 2, bytes: 6, allocs: 5, characters: 123},
		{nanoseconds: 5, bytes: 3, allocs: 2, characters: 123},
		{nanoseconds: 3, bytes: 5, allocs: 7, characters: 123},
		{nanoseconds: 4, bytes: 4, allocs: 1, characters: 123},
	}
	got, err := median(points)
	if err != nil {
		t.Fatal(err)
	}
	if want := (measurement{nanoseconds: 4, bytes: 4, allocs: 4, characters: 123}); got != want {
		t.Fatalf("median: got %+v, want %+v", got, want)
	}
	points[0].characters = 0
	if _, err := median(points); err == nil {
		t.Fatal("missing code count must fail")
	}
	points[0].characters = 124
	if _, err := median(points); err == nil {
		t.Fatal("mixed source sizes must fail rather than being averaged")
	}
	points[0].characters = 123
	points[0].complexity = -1
	if _, err := median(points); err == nil {
		t.Fatal("missing complexity must fail")
	}
	points[0].complexity = 1
	if _, err := median(points); err == nil {
		t.Fatal("mixed complexity scores must fail rather than being averaged")
	}
	for index := range points {
		points[index].complexity = 3
	}
	got, err = median(points)
	if err != nil {
		t.Fatal(err)
	}
	if got.complexity != 3 {
		t.Fatalf("complexity: got %d, want 3", got.complexity)
	}
}
