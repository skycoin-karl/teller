package scanner

import (
	"container/list"
	"fmt"
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

			//s.process()
			s.test_process()
		}
	}()
}

func (s *Scanner) test_process() {
	s.Lock()
	defer s.Unlock()

	e := s.work.Front()

	defer println("-------------")

	for i := 0; i < s.work.Len(); i++ {
		if e == nil {
			return
		}

		w := e.Value.(*types.Work)

		fmt.Println(w.Request.Drop)

		if w.Request.Drop == types.Drop("1GXoqycXrtguntcAHPycuYw9R6xUqpTVDb") {
			w.Request.Metadata.Status = types.SEND
			w.Return(nil)
			s.work.Remove(e)
			e = e.Next()
			continue
		}

		e = e.Next()
	}
}

func (s *Scanner) process() {
	s.Lock()
	defer s.Unlock()

	// get first item
	e := s.work.Front()

	var (
		w       *types.Work
		balance float64
		err     error
	)

	for i := 0; i < s.work.Len(); i++ {
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
			e = e.Next()
			continue
		}

		// get balance of drop
		if balance, err = s.dropper.GetBalance(
			w.Request.Currency,
			w.Request.Drop,
		); err != nil {
			w.Return(err)
			s.work.Remove(e)
			e = e.Next()
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
		e = e.Next()
	}
}

func (s *Scanner) Handle(request *types.Request) chan *types.Result {
	s.Lock()
	defer s.Unlock()

	result := make(chan *types.Result, 1)
	s.work.PushFront(&types.Work{request, result})
	return result
}
