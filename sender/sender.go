package sender

import (
	"container/list"
	"errors"
	"sync"
	"time"

	"github.com/skycoin-karl/teller/dropper"
	"github.com/skycoin-karl/teller/skycoin"
	"github.com/skycoin-karl/teller/types"
	"github.com/skycoin/skycoin/src/api/cli"
)

type Sender struct {
	sync.Mutex

	skycoin *skycoin.Connection
	dropper *dropper.Dropper
	work    *list.List
}

var ErrZeroBalance = errors.New("sender got drop with zero balance")

func NewSender(s *skycoin.Connection, d *dropper.Dropper) (*Sender, error) {
	return &Sender{
		skycoin: s,
		dropper: d,
		work:    list.New().Init(),
	}, nil
}

func (s *Sender) Start() {
	go func() {
		for {
			// TODO: tick
			if s.work.Len() == 0 {
				<-time.After(time.Second * 3)
			} else {
				<-time.After(time.Second)
			}

			s.process()
		}
	}()
}

func (s *Sender) process() {
	s.Lock()
	defer s.Unlock()

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

		// sender shouldn't have requests with zero balance
		if balance == 0.0 {
			w.Return(ErrZeroBalance)
			s.work.Remove(e)
			continue
		}

		to := []cli.SendAmount{{
			Addr:  string(w.Request.Address),
			Coins: s.dropper.GetValue(w.Request.Currency, balance),
		}}

		// create sky transaction
		tx, err := cli.CreateRawTx(
			s.skycoin.Client,
			s.skycoin.Wallet,
			s.fromAddrs(),
			s.fromChangeAddr(),
			to,
		)
		if err != nil {
			w.Return(err)
			s.work.Remove(e)
			continue
		}

		// inject and get txId
		txId, err := s.skycoin.Client.InjectTransaction(tx)
		if err != nil {
			w.Return(err)
			s.work.Remove(e)
			continue
		}

		// next step is monitor service
		w.Request.Metadata.TxId = txId
		w.Request.Metadata.Status = types.CONFIRM
		w.Return(nil)

		// remove from queue
		s.work.Remove(e)

		// get next item in queue
		e = e.Next()
	}
}

func (s *Sender) fromAddrs() []string {
	return nil
}

func (s *Sender) fromChangeAddr() string {
	return ""
}

func (s *Sender) Handle(request *types.Request) chan *types.Result {
	result := make(chan *types.Result)

	defer func() {
		w := &types.Work{request, result}

		s.Lock()
		s.work.PushFront(w)
		s.Unlock()
	}()

	return result
}
