package dropper

import (
	"io/ioutil"
	"path/filepath"

	"github.com/davecgh/go-spew/spew"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcutil"
	"github.com/skycoin-karl/teller/types"
)

type BTCConnection struct {
	client  *rpcclient.Client
	account string
	testnet bool
}

func NewBTCConnection(config *types.Config) (*BTCConnection, error) {
	// get tls certs for websocket connection
	certs, err := ioutil.ReadFile(
		filepath.Join(
			btcutil.AppDataDir("btcwallet", false),
			"rpc.cert",
		),
	)
	if err != nil {
		return nil, err
	}

	// connect to btc node
	client, err := rpcclient.New(
		&rpcclient.ConnConfig{
			Host:         config.Dropper.BTC.Node,
			Endpoint:     "ws",
			User:         config.Dropper.BTC.User,
			Pass:         config.Dropper.BTC.Pass,
			Certificates: certs,
		},
		nil,
	)
	if err != nil {
		return nil, err
	}

	// get list of all accounts
	accounts, err := client.ListAccounts()
	if err != nil {
		return nil, err
	}

	// check if at least one is for teller-json
	exists := false
	for account, _ := range accounts {
		if account == config.Dropper.BTC.Account {
			exists = true
		}
	}

	// account for teller doesn't exist, need to create a new one
	if !exists {
		// authenticate with the wallet passphrase
		err = client.WalletPassphrase(config.Dropper.BTC.Account, 2)
		if err != nil {
			return nil, err
		}

		// create new account for generating addresses
		err = client.CreateNewAccount(config.Dropper.BTC.Account)
		if err != nil {
			return nil, err
		}
	}

	a, err := client.GetBalance(config.Dropper.BTC.Account)
	if err != nil {
		return nil, err
	}

	println(a)

	return &BTCConnection{
		client:  client,
		account: config.Dropper.BTC.Account,
		testnet: config.Dropper.BTC.Testnet,
	}, nil
}

func (c *BTCConnection) Generate() (types.Drop, error) {
	addr, err := c.client.GetNewAddress(c.account)
	if err != nil {
		return "", err
	}

	return types.Drop(addr.EncodeAddress()), nil
}

func (c *BTCConnection) Balance(drop types.Drop) (float64, error) {
	var params *chaincfg.Params
	if c.testnet {
		params = &chaincfg.TestNet3Params
	} else {
		params = &chaincfg.MainNetParams
	}

	// convert address string to btc struct
	addr, err := btcutil.DecodeAddress(string(drop), params)
	if err != nil {
		return 0, nil
	}

	// get unspents of address
	unspents, err := c.client.ListUnspentMinMaxAddresses(
		1,                       // min confirmations
		999999,                  // max confirmations
		[]btcutil.Address{addr}, // only checking one address
	)
	if err != nil {
		return 0, nil
	}

	scs := spew.ConfigState{Indent: "\t"}
	scs.Dump(unspents)

	var sum float64
	for _, res := range unspents {
		if res.Spendable {
			sum += res.Amount
		}
	}

	return sum, nil
}

func (c *BTCConnection) Connected() (bool, error) {
	return !c.client.Disconnected(), c.client.Ping()
}

func (c *BTCConnection) Stop() error {
	c.client.Shutdown()
	c.client.WaitForShutdown()
	return nil
}
