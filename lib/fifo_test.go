package lib

import "reflect"
import "testing"

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
	if !reflect.DeepEqual(s, []portalIndex{1}) {
		t.Errorf("Reversing [1] should return [1]")
	}

	s = []portalIndex{1, 2}
	reverse(s)
	if !reflect.DeepEqual(s, []portalIndex{2, 1}) {
		t.Errorf("Reversing [1, 2] should return [2, 1]")
	}

	s = []portalIndex{1, 2, 3}
	reverse(s)
	if !reflect.DeepEqual(s, []portalIndex{3, 2, 1}) {
		t.Errorf("Reversing [1, 2, 3] should return [3, 2, 1]")
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
	for i := 0; i < 5; i++ {
		if output[i] != portalIndex(2*i+1) {
			t.Errorf("Expected %d-th value to be %d, got %d", i, 2*i+1, output[i])
		}
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
