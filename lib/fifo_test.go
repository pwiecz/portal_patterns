package lib

import "testing"

func TestFifoEmpty(t *testing.T) {
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
	for i := 0; i < 5; i++ {
		if output[i] != portalIndex(2*i+1) {
			t.Errorf("Expected %d-th value to be %d, got %d", i, 2*i+1, output[i])
		}
	}
	if !q.Empty() {
		t.Errorf("Expected queue to be empty at the end")
	}
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Excepted Dequeue on empty queue to cause panic")
		}
	}()
	q.Dequeue()
}
