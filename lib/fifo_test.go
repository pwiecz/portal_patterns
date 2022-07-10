package lib

import (
	"testing"

	"golang.org/x/exp/slices"
)

func TestReverse(t *testing.T) {
	s := []portalIndex(nil)
	reverse(s)
	if s != nil {
		t.Errorf("Reversing nil slice should not change it")
	}

	s = []portalIndex{}
	reverse(s)
	if len(s) != 0 {
		t.Errorf("Reversing empty slice should keep it empty")
	}

	s = []portalIndex{1}
	reverse(s)
	if !slices.Equal(s, []portalIndex{1}) {
		t.Errorf("Expected [1], got %v", s)
	}

	s = []portalIndex{1, 2}
	reverse(s)
	if !slices.Equal(s, []portalIndex{2, 1}) {
		t.Errorf("Expected [2 1], got %v", s)
	}

	s = []portalIndex{1, 2, 3}
	reverse(s)
	if !slices.Equal(s, []portalIndex{3, 2, 1}) {
		t.Errorf("Expected [3 2 1], got %v", s)
	}
}

func TestFifoEnqueueDequeue(t *testing.T) {
	q := fifo{}
	if !q.Empty() {
		t.Errorf("Expected queue to be initially empty")
	}
	q.Enqueue(1)
	q.Enqueue(3)
	q.Enqueue(5)
	output := []portalIndex{q.Dequeue(), q.Dequeue()}
	q.Enqueue(7)
	q.Enqueue(9)
	output = append(output, q.Dequeue(), q.Dequeue(), q.Dequeue())
	if !slices.Equal(output, []portalIndex{1, 3, 5, 7, 9}) {
		t.Errorf("Expected [1 3 5 7 9], got %v", output)
	}
	if !q.Empty() {
		t.Errorf("Expected queue to be empty at the end")
	}
}

func TestFifoReset(t *testing.T) {
	q := fifo{}
	if !q.Empty() {
		t.Errorf("Expected queue to be initially empty")
	}
	q.Enqueue(1)
	q.Reset()
	q.Enqueue(3)
	q.Enqueue(5)
	q.Enqueue(7)
	output := []portalIndex{q.Dequeue()}
	q.Enqueue(9)
	output = append(output, q.Dequeue(), q.Dequeue(), q.Dequeue())
	if !slices.Equal(output, []portalIndex{3, 5, 7, 9}) {
		t.Errorf("Expected [3 5 7 9], got %v", output)
	}
	if !q.Empty() {
		t.Errorf("Expected queue to be empty at the end")
	}
}

func TestFifoEmpty(t *testing.T) {
	q := fifo{}
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Excepted Dequeue on empty queue to cause panic")
		}
	}()
	q.Dequeue()
}
