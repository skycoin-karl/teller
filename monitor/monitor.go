package monitor

import (
	"container/list"
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
			<-time.After(time.Second * 5)

			m.process()
		}
	}()
}

func (m *Monitor) process() {
	m.Lock()
	defer m.Unlock()

	e := m.work.Front()
	var w *types.Work

	for i := 0; i < m.work.Len(); i++ {
		if e == nil {
			return
		}

		w = e.Value.(*types.Work)

		// get sky transaction
		tx, err := m.skycoin.Client.GetTransactionByID(w.Request.Metadata.TxId)
		if err != nil {
			w.Return(err)
			m.work.Remove(e)
			e = e.Next()
			continue
		}

		// if not confirmed, move to next work
		if !tx.Transaction.Status.Confirmed {
			e = e.Next()
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
	m.Lock()
	defer m.Unlock()

	result := make(chan *types.Result, 1)
	m.work.PushFront(&types.Work{request, result})
	return result
}
