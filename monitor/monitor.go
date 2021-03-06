package monitor

import (
	"container/list"
	"log"
	"os"
	"sync"
	"time"

	"github.com/skycoin-karl/teller/skycoin"
	"github.com/skycoin-karl/teller/types"
)

type Monitor struct {
	sync.Mutex

	config  *types.Config
	skycoin *skycoin.Connection
	logger  *log.Logger
	work    *list.List
	stop    chan struct{}
	stopped bool
}

func NewMonitor(c *types.Config, sky *skycoin.Connection) (*Monitor, error) {
	return &Monitor{
		config:  c,
		skycoin: sky,
		logger:  log.New(os.Stdout, types.LOG_MONITOR, types.LOG_FLAGS),
		work:    list.New().Init(),
		stop:    make(chan struct{}),
	}, nil
}

func (m *Monitor) Stop() {
	m.stop <- struct{}{}
	m.logger.Println("stopped")
}

func (m *Monitor) Start() {
	m.logger.Println("started")
	go func() {
		for {
			<-time.After(time.Second * time.Duration(m.config.Monitor.Tick))

			m.logger.Printf("[%d]\n", m.work.Len())

			select {
			case <-m.stop:
				m.stopped = true
				return
			default:
				m.process()
			}
		}
	}()
}

func (m *Monitor) process() {
	m.Lock()
	defer m.Unlock()

	for e := m.work.Front(); e != nil; e = e.Next() {
		w := e.Value.(*types.Work)

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
	}
}

func (m *Monitor) Handle(request *types.Request) chan *types.Result {
	m.Lock()
	defer m.Unlock()

	result := make(chan *types.Result, 1)
	m.work.PushBack(&types.Work{request, result})
	return result
}
