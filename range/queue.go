package byterange

import (
	"sync"
)

type JobQueue struct {
	items []*job
	sync.Mutex
}

func (q *JobQueue) Len() int { return len(q.items) }

func (q *JobQueue) Less(i, j int) bool {
	return q.items[i].index < q.items[j].index
}

func (q *JobQueue) Swap(i, j int) {
	q.Lock()
	defer q.Unlock()
	q.items[i], q.items[j] = q.items[j], q.items[i]
}

func (q *JobQueue) Push(item *job) {
	q.Lock()
	defer q.Unlock()
	q.items = append(q.items, item)
}

func (q *JobQueue) PushFront(item *job) {
	q.Lock()
	defer q.Unlock()

	q.items = append(q.items, nil)
	copy(q.items[1:], q.items)
	q.items[0] = item
}

func (q *JobQueue) Pop() (*job, bool) {
	q.Lock()
	defer q.Unlock()

	if len(q.items) == 0 {
		return nil, false
	}

	item := q.items[0]
	q.items = q.items[1:]

	return item, true
}
