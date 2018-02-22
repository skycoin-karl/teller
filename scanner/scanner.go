package scanner

import (
	"container/list"
	"sync"
	"time"

	"github.com/davecgh/go-spew/spew"

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
			if s.work.Len() == 0 {
				<-time.After(time.Second * 3)
			} else {
				<-time.After(time.Second)
			}

			println("scanner scan")
			s.process()
		}
	}()
}

func (s *Scanner) process() {
	s.Lock()
	defer s.Unlock()

	scs := spew.ConfigState{Indent: "\t"}
	scs.Dump(s.work)

	// get first item
	e := s.work.Front()

	var (
		w       *types.Work
		balance float64
		err     error
	)

	for {
		// nothing left in queue
		if e == nil {
			return
		}
		w = e.Value.(*types.Work)

		// check if expired
		if w.Request.Metadata.Expired() {
			w.Request.Metadata.Status = types.EXPIRED
			w.Return(nil)
			s.work.Remove(e)
			continue
		}

		// get balance of drop
		if balance, err = s.dropper.GetBalance(
			w.Request.Currency,
			w.Request.Drop,
		); err != nil {
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

		// remove from queue
		s.work.Remove(e)

		// get next item in queue
		e = e.Next()
	}
}

func (s *Scanner) Handle(request *types.Request) chan *types.Result {
	result := make(chan *types.Result)

	defer func() {
		w := &types.Work{request, result}

		s.Lock()
		s.work.PushFront(w)
		s.Unlock()
	}()

	return result
}
