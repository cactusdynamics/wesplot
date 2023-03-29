package wesplot

import (
	"container/ring"

	"golang.org/x/exp/constraints"
)

type Number interface {
	constraints.Float | constraints.Integer
}

func Filter[T any](slice []T, predicate func(T) bool) []T {
	filtered := make([]T, 0, len(slice))
	for _, elem := range slice {
		if predicate(elem) {
			filtered = append(filtered, elem)
		}
	}
	return filtered
}

func Min[T Number](a T, b T) T {
	if a > b {
		return b
	}

	return a
}

// Ring taken from https://github.com/Shopify/mybench/blob/main/ring.go
// This is not a particularly efficient implementation (as it allocates on
// read), but a good first starting point.
//
// The mutex has been removed from the upstream implementation because the
// DataBroadcaster has a single mutex governing both data buffering and channel
// management. Having another mutex here is both unnecessary and a potential for
// deadlocks.
//
// Copyright 2022-present, Shopify Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

// A terrible implementation of a ring, based on the Golang ring which is not
// thread-safe nor offers a nice API.
//
// I can't believe there are no simple ring buffer data structure in Golang,
// with generics.
type ThreadUnsafeRing[T any] struct {
	capacity int
	ring     *ring.Ring
}

func NewRing[T any](capacity int) *ThreadUnsafeRing[T] {
	return &ThreadUnsafeRing[T]{
		capacity: capacity,
		ring:     ring.New(capacity),
	}
}

func (r *ThreadUnsafeRing[T]) Push(data T) {
	r.ring = r.ring.Next()
	r.ring.Value = data
}

func (r *ThreadUnsafeRing[T]) ReadAllOrdered() []T {
	arr := make([]T, 0, r.capacity)

	earliest := r.ring

	for earliest.Prev() != nil && earliest.Prev() != r.ring && earliest.Prev().Value != nil {
		earliest = earliest.Prev()
	}

	for earliest != r.ring {
		arr = append(arr, earliest.Value.(T))
		earliest = earliest.Next()
	}

	if earliest.Value == nil {
		return arr
	}

	arr = append(arr, earliest.Value.(T))

	return arr
}
