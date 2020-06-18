package lib

func reverse(s []portalIndex) {
	for i := len(s)/2 - 1; i >= 0; i-- {
		opp := len(s) - 1 - i
		s[i], s[opp] = s[opp], s[i]
	}
}

type fifo struct {
	front, back []portalIndex
}

func (f *fifo) Reset() {
	f.front = f.back[:0]
	f.back = f.back[:0]
}
func (f *fifo) Empty() bool {
	return len(f.front)+len(f.back) == 0
}
func (f *fifo) Enqueue(p portalIndex) {
	f.back = append(f.back, p)
}
func (f *fifo) Dequeue() portalIndex {
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
