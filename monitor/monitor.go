package monitor

import (
	"container/list"
	"fmt"
	"sync"
	"time"

	"github.com/skycoin-karl/teller/skycoin"
	"github.com/skycoin-karl/teller/types"
)

type Monitor struct {
	sync.Mutex

	work    *list.List
	skycoin *skycoin.Connection
}

func NewMonitor(sky *skycoin.Connection) (*Monitor, error) {
	return &Monitor{
		skycoin: sky,
		work:    list.New().Init(),
	}, nil
}

func (m *Monitor) Start() {
	go func() {
		for {
			// TODO: tick
			if m.work.Len() == 0 {
				<-time.After(time.Second * 3)
			} else {
				<-time.After(time.Second)
			}

			m.process()
		}
	}()
}

func (m *Monitor) process() {
	m.Lock()
	defer m.Unlock()

	e := m.work.Front()
	var w *types.Work

	for {
		if e == nil {
			return
		}
		w = e.Value.(*types.Work)

		fmt.Println(w.Request)

		// get sky transaction
		tx, err := m.skycoin.Client.GetTransactionByID(w.Request.Metadata.TxId)
		if err != nil {
			w.Return(err)
			m.work.Remove(e)
			continue
		}

		// if not confirmed, move to next work
		if !tx.Transaction.Status.Confirmed {
			continue
		}

		// all done
		w.Request.Metadata.Status = types.DONE
		w.Return(nil)

		// remove from queue
		m.work.Remove(e)

		// get next item in queue
		e = e.Next()
	}
}

func (m *Monitor) Handle(request *types.Request) chan *types.Result {
	result := make(chan *types.Result)

	defer func() {
		w := &types.Work{request, result}

		m.Lock()
		m.work.PushFront(w)
		m.Unlock()
	}()

	return result
}
