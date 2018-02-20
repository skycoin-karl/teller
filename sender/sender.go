package sender

import (
	"errors"

	"github.com/skycoin-karl/teller/dropper"
	"github.com/skycoin-karl/teller/skycoin"
	"github.com/skycoin-karl/teller/types"
	"github.com/skycoin/skycoin/src/api/cli"
)

type Sender struct {
	skycoin *skycoin.Connection
	dropper *dropper.Dropper
}

var ErrZeroBalance = errors.New("sender got drop with zero balance")

func NewSender(s *skycoin.Connection, d *dropper.Dropper) (*Sender, error) {
	return &Sender{s, d}, nil
}

func (s *Sender) fromAddrs() []string {
	return nil
}

func (s *Sender) fromChangeAddr() string {
	return ""
}

func (s *Sender) Handle(r *types.Request) error {
	// get balance of drop
	balance, err := s.dropper.GetBalance(r.Currency, r.Drop)
	if err != nil {
		return err
	}

	// sender shouldn't have a request with zero balance
	if balance == 0.0 {
		return ErrZeroBalance
	}

	// calculate value and sendamount
	to := []cli.SendAmount{
		{
			Addr:  string(r.Address),
			Coins: s.dropper.GetValue(r.Currency, balance),
		},
	}

	// create sky transaction
	tx, err := cli.CreateRawTx(
		s.skycoin.Client,
		s.skycoin.Wallet,
		s.fromAddrs(),
		s.fromChangeAddr(),
		to,
	)
	if err != nil {
		return err
	}

	// inject and get txId
	txId, err := s.skycoin.Client.InjectTransaction(tx)
	if err != nil {
		return err
	}

	// next step is monitor service
	r.Metadata.TxId = txId
	r.Metadata.Status = types.CONFIRM
	return nil
}
