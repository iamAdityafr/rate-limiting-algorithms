package main

type Deque[T any] struct {
	items []T
	front int
	size  int
}

func NewDeque[T any]() *Deque[T] {
	return &Deque[T]{
		items: make([]T, 16),
		front: 0,
		size:  0,
	}
}

func (d *Deque[T]) PushBack(item T) {
	if d.size == len(d.items) {
		d.resize()
	}
	back := (d.front + d.size) % len(d.items)
	d.items[back] = item
	d.size++
}

func (d *Deque[T]) PopFront() (T, bool) {
	var data T
	if d.size == 0 {
		return data, false
	}
	item := d.items[d.front]
	d.items[d.front] = data
	d.front = (d.front + 1) % len(d.items)
	d.size--
	return item, true
}

func (d *Deque[T]) PeekFront() (T, bool) {
	var data T
	if d.size == 0 {
		return data, false
	}
	return d.items[d.front], true
}

func (d *Deque[T]) Size() int {
	return d.size
}

func (d *Deque[T]) IsEmpty() bool {
	return d.size == 0
}

func (d *Deque[T]) resize() {
	newCapacity := len(d.items) * 2
	newItems := make([]T, newCapacity)
	for i := 0; i < d.size; i++ {
		newItems[i] = d.items[(d.front+i)%len(d.items)]
	}
	d.items = newItems
	d.front = 0
}
