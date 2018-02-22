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

	dropper *dropper.Dropper
	work    *list.List
}

func NewScanner(drpr *dropper.Dropper) (*Scanner, error) {
	return &Scanner{
		dropper: drpr,
		work:    list.New().Init(),
	}, nil
}

func (s *Scanner) Start() {
	go func() {
		for {
			// TODO: tick
			<-time.After(time.Second * 1)

			s.process()
		}
	}()
}

func (s *Scanner) process() {
	s.Lock()
	defer s.Unlock()

	for e := s.work.Front(); e != nil; e = e.Next() {
		w := e.Value.(*types.Work)

		// check if expired
		if w.Request.Metadata.Expired() {
			w.Request.Metadata.Status = types.EXPIRED
			w.Return(nil)
			s.work.Remove(e)
			continue
		}

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

		// nothing has happened
		if balance == 0.0 {
			continue
		}

		// balance detected, so next step is sender
		w.Request.Metadata.Status = types.SEND
		w.Return(nil)
		s.work.Remove(e)
	}
}

func (s *Scanner) Handle(request *types.Request) chan *types.Result {
	s.Lock()
	defer s.Unlock()

	result := make(chan *types.Result, 1)
	s.work.PushFront(&types.Work{request, result})
	return result
}
