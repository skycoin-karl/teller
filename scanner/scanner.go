package scanner

import (
	"container/list"
	"sync"
	"time"

	"github.com/skycoin-karl/teller/dropper"
	"github.com/skycoin-karl/teller/types"
)

type Scanner struct {
	sync.Mutex

	config  *types.Config
	dropper *dropper.Dropper
	work    *list.List
	stop    chan struct{}
}

func NewScanner(conf *types.Config, drpr *dropper.Dropper) (*Scanner, error) {
	return &Scanner{
		dropper: drpr,
		config:  conf,
		work:    list.New().Init(),
		stop:    make(chan struct{}),
	}, nil
}

func (s *Scanner) Stop() { s.stop <- struct{}{} }

func (s *Scanner) Start() {
	go func() {
		for {
			<-time.After(time.Second * time.Duration(s.config.Scanner.Tick))

			select {
			case <-s.stop:
				return
			default:
				s.process()
			}
		}
	}()
}

func (s *Scanner) process() {
	s.Lock()
	defer s.Unlock()

	for e := s.work.Front(); e != nil; e = e.Next() {
		w := e.Value.(*types.Work)

		// check if expired
		if w.Request.Metadata.Expired(s.config.Scanner.Expiration) {
			w.Request.Metadata.Status = types.EXPIRED
			w.Return(nil)
			s.work.Remove(e)
			continue
		}

		println("scanning " + string(w.Request.Drop))

		// get balance of drop
		balance, err := s.dropper.GetBalance(
			w.Request.Currency,
			w.Request.Drop,
		)
		if err != nil {
			w.Return(err)
			s.work.Remove(e)
			continue
		}

		// user made a deposit
		if balance != 0.0 {
			w.Request.Metadata.Status = types.SEND
			w.Return(nil)
			s.work.Remove(e)
		}
	}
}

func (s *Scanner) Handle(request *types.Request) chan *types.Result {
	s.Lock()
	defer s.Unlock()

	result := make(chan *types.Result, 1)
	s.work.PushBack(&types.Work{request, result})
	return result
}
