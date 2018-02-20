package scanner

import (
	"time"

	"github.com/skycoin-karl/teller/dropper"
	"github.com/skycoin-karl/teller/types"
)

type Scanner struct {
	dropper *dropper.Dropper
}

func NewScanner(drpr *dropper.Dropper) (*Scanner, error) {
	return &Scanner{drpr}, nil
}

func (s *Scanner) Handle(r *types.Request) error {
	// continue scanning until request expires
	for !r.Metadata.Expired() {
		// wait before scanning
		<-time.After(time.Minute * 5)

		// get balance of drop
		balance, err := s.dropper.GetBalance(r.Currency, r.Drop)
		if err != nil {
			return err
		}

		// nothing has happened
		if balance == 0.0 {
			continue
		}

		// there is a balance, so next step is sender
		r.Metadata.Status = types.SEND
		return nil
	}

	// if loop breaks, then request has expired
	r.Metadata.Status = types.EXPIRED

	return nil
}
