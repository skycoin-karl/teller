package dropper

import (
	"io/ioutil"
	"path/filepath"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcutil"
	"github.com/skycoin-karl/teller/types"
)

const ACCOUNT = "teller"

type BTCConnection struct {
	client *rpcclient.Client
}

func NewBTCConnection(p string) (*BTCConnection, error) {
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
			Host:         p,
			Endpoint:     "ws",
			User:         ACCOUNT,
			Pass:         ACCOUNT,
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
		if account == ACCOUNT {
			exists = true
		}
	}

	// account for teller-json doesn't exist, need to create a new one
	if !exists {
		// authenticate with the wallet passphrase
		err = client.WalletPassphrase(ACCOUNT, 2)
		if err != nil {
			return nil, err
		}

		// create new account for generating addresses
		err = client.CreateNewAccount(ACCOUNT)
		if err != nil {
			return nil, err
		}
	}

	return &BTCConnection{client: client}, nil
}

func (c *BTCConnection) Generate() (types.Drop, error) {
	addr, err := c.client.GetNewAddress(ACCOUNT)
	if err != nil {
		return "", err
	}

	return types.Drop(addr.EncodeAddress()), nil
}

func (c *BTCConnection) Balance(drop types.Drop) (float64, error) {
	// convert address string to btc struct
	addr, err := btcutil.DecodeAddress(string(drop), &chaincfg.MainNetParams)
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
