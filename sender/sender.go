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

	config  *types.Config
	skycoin *skycoin.Connection
	dropper *dropper.Dropper
	work    *list.List
	stop    chan struct{}
}

var ErrZeroBalance = errors.New("sender got drop with zero balance")

func NewSender(c *types.Config, s *skycoin.Connection, d *dropper.Dropper) (*Sender, error) {
	return &Sender{
		config:  c,
		skycoin: s,
		dropper: d,
		work:    list.New().Init(),
		stop:    make(chan struct{}),
	}, nil
}

func (s *Sender) Stop() { s.stop <- struct{}{} }

func (s *Sender) Start() {
	go func() {
		for {
			<-time.After(time.Second * time.Duration(s.config.Sender.Tick))

			select {
			case <-s.stop:
				return
			default:
				s.process()
			}
		}
	}()
}

func (s *Sender) process() {
	s.Lock()
	defer s.Unlock()

	for e := s.work.Front(); e != nil; e = e.Next() {
		w := e.Value.(*types.Work)

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
	}
}

func (s *Sender) fromAddrs() []string {
	return nil
}

func (s *Sender) fromChangeAddr() string {
	return ""
}

func (s *Sender) Handle(request *types.Request) chan *types.Result {
	s.Lock()
	defer s.Unlock()

	result := make(chan *types.Result, 1)
	s.work.PushBack(&types.Work{request, result})
	return result
}
