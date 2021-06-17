package main

import "sync"

func reverse(s []func()) {
	for i := len(s)/2 - 1; i >= 0; i-- {
		opp := len(s) - 1 - i
		s[i], s[opp] = s[opp], s[i]
	}
}

type TaskQueue struct {
	front, back []func()
	mutex       sync.Mutex
}

func (f *TaskQueue) Reset() {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	f.front = f.front[:0]
	f.back = f.back[:0]
}
func (f *TaskQueue) Empty() bool {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	return len(f.front)+len(f.back) == 0
}
func (f *TaskQueue) Enqueue(p func()) {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	f.back = append(f.back, p)
}
func (f *TaskQueue) Dequeue() func() {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	lenFirst := len(f.front)
	if lenFirst == 0 {
		lenFirst = len(f.back)
		if lenFirst == 0 {
			panic("Empty queue")
		}
		f.front, f.back = f.back, f.front
		reverse(f.front)
	}
	r := f.front[lenFirst-1]
	f.front = f.front[:lenFirst-1]
	return r
}

func (f *TaskQueue) Size() int {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	return len(f.front) + len(f.back)
}
