package queue

import (
	"container/list"

	"github.com/skycoin-karl/teller/types"
)

type Queue struct {
	list *list.List
}

func NewQueue() *Queue {
	return &Queue{
		list: list.New().Init(),
	}
}

func (q *Queue) Len() int { return q.list.Len() }

func (q *Queue) Push(r *types.Request) {
	q.list.PushFront(r)
}

func (q *Queue) Pop() *types.Request {

}
