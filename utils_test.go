package wesplot

import (
	"math"
	"reflect"
	"testing"
)

func TestFilter(t *testing.T) {
	t.Run("empty slice", func(t *testing.T) {
		var input []int = nil
		pred := func(int) bool { return true }
		got := Filter(input, pred)
		want := []int{}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("Filter(%v) = %v, want %v", input, got, want)
		}
	})

	t.Run("no matches", func(t *testing.T) {
		input := []int{1, 2, 3}
		pred := func(x int) bool { return x > 10 }
		got := Filter(input, pred)
		want := []int{}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("Filter(%v) = %v, want %v", input, got, want)
		}
	})

	t.Run("all match", func(t *testing.T) {
		input := []int{1, 2, 3}
		pred := func(x int) bool { return x > 0 }
		got := Filter(input, pred)
		want := []int{1, 2, 3}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("Filter(%v) = %v, want %v", input, got, want)
		}
	})

	t.Run("partial match", func(t *testing.T) {
		input := []int{1, 2, 3}
		pred := func(x int) bool { return x%2 == 1 }
		got := Filter(input, pred)
		want := []int{1, 3}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("Filter(%v) = %v, want %v", input, got, want)
		}
	})
}

func TestMin(t *testing.T) {
	if got := Min(5, 3); got != 3 {
		t.Fatalf("Min(5,3) = %v, want 3", got)
	}

	if got := Min(4, 4); got != 4 {
		t.Fatalf("Min(4,4) = %v, want 4", got)
	}

	a := math.NaN()
	if got := Min(a, 1.0); !math.IsNaN(got) {
		t.Fatalf("Min(NaN,1.0) = %v, want NaN", got)
	}

	if got := Min(1.0, a); got != 1.0 {
		t.Fatalf("Min(1.0,NaN) = %v, want 1.0", got)
	}
}

func TestThreadUnsafeRing(t *testing.T) {
	// tests run; skip individual failing cases if needed

	t.Run("capacity 1 overwrite", func(t *testing.T) {
		r := NewRing[int](1)
		r.Push(1)
		r.Push(2)
		got := r.ReadAllOrdered()
		want := []int{2}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %v want %v", got, want)
		}
	})

	t.Run("partial fill preserved order", func(t *testing.T) {
		r := NewRing[int](3)
		r.Push(10)
		r.Push(20)
		got := r.ReadAllOrdered()
		want := []int{10, 20}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %v want %v", got, want)
		}
	})

	t.Run("exact capacity order", func(t *testing.T) {
		r := NewRing[int](3)
		r.Push(1)
		r.Push(2)
		r.Push(3)
		got := r.ReadAllOrdered()
		want := []int{1, 2, 3}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %v want %v", got, want)
		}
	})

	t.Run("wraparound preserves newest", func(t *testing.T) {
		r := NewRing[int](3)
		r.Push(1)
		r.Push(2)
		r.Push(3)
		r.Push(4)
		got := r.ReadAllOrdered()
		want := []int{2, 3, 4}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %v want %v", got, want)
		}
	})

	t.Run("multiple wraparound exact multiples", func(t *testing.T) {
		r := NewRing[int](3)
		for i := 1; i <= 6; i++ {
			r.Push(i)
		}
		got := r.ReadAllOrdered()
		want := []int{4, 5, 6}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %v want %v", got, want)
		}

		// another exact multiple (3 * 3)
		for i := 7; i <= 9; i++ {
			r.Push(i)
		}
		got = r.ReadAllOrdered()
		want = []int{7, 8, 9}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %v want %v", got, want)
		}
	})

	t.Run("multiple wraparound overflow", func(t *testing.T) {
		r := NewRing[int](3)
		for i := 1; i <= 7; i++ {
			r.Push(i)
		}
		got := r.ReadAllOrdered()
		want := []int{5, 6, 7}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %v want %v", got, want)
		}
	})

	t.Run("read on empty returns empty", func(t *testing.T) {
		r := NewRing[int](3)
		got := r.ReadAllOrdered()
		if len(got) != 0 {
			t.Fatalf("expected empty slice, got %v", got)
		}
	})

	t.Run("pointer nil handling", func(t *testing.T) {
		r := NewRing[*int](3)
		r.Push(nil)
		a := 5
		r.Push(&a)
		b := 10
		r.Push(&b)
		got := r.ReadAllOrdered()
		if len(got) != 3 {
			t.Fatalf("expected 3 entries (including nil), got %v", got)
		}
		if got[0] != nil || *got[1] != 5 || *got[2] != 10 {
			t.Fatalf("pointer ring got %v", got)
		}
	})

	t.Run("zero capacity panics", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Fatalf("expected panic for zero capacity ring")
			}
		}()
		r := NewRing[int](0)
		r.Push(1)
	})
}
