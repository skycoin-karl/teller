package skycoin

import (
	"fmt"

	"github.com/skycoin/skycoin/src/api/webrpc"
	"github.com/skycoin/skycoin/src/wallet"
)

type Connection struct {
	Wallet *wallet.Wallet
	Client *webrpc.Client
}

func NewConnection(addr, seed string) (*Connection, error) {
	c := &webrpc.Client{Addr: addr}
	if s, err := c.GetStatus(); err != nil {
		return nil, err
	} else if !s.Running {
		return nil, fmt.Errorf("node isn't running at %s", addr)
	}

	w, err := wallet.NewWallet(
		"teller",
		wallet.Options{
			Coin:  wallet.CoinTypeSkycoin,
			Label: "teller",
			Seed:  seed,
		},
	)
	if err != nil {
		return nil, err
	}

	return &Connection{Wallet: w, Client: c}, nil
}
