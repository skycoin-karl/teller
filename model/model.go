package model

import (
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"os"

	"github.com/skycoin-karl/teller/types"
	"github.com/skycoin/skycoin/src/cipher"
)

var (
	ErrUnknownStatus   = errors.New("unknown status type")
	ErrInvalidFilename = errors.New("invalid filename in db dir")
)

type Model struct {
	path    string
	errs    chan error
	Scanner types.Service
	Sender  types.Service
	Monitor types.Service
}

func NewModel(path string, scnr, sndr, mntr types.Service) (*Model, error) {
	m := &Model{
		path:    path,
		errs:    make(chan error),
		Scanner: scnr,
		Sender:  sndr,
		Monitor: mntr,
	}

	go m.logger()

	// make sure db dir exists
	_, err := os.Stat(m.path)
	if err != nil {
		return nil, err
	}

	// get list of files in db dir
	files, err := ioutil.ReadDir(m.path)
	if err != nil {
		return nil, err
	}

	// for each .json file in the db dir
	for _, file := range files {
		// create a slice of requests contained in file
		requests, err := m.load(file.Name())
		if err != nil {
			return nil, err
		}

		// inject each request into the proper service
		for _, request := range requests {
			go func() {
				if err := m.Handle(request); err != nil {
					m.errs <- err
				}
			}()
		}
	}

	return m, nil
}

func (m *Model) logger() {
	for {
		err := <-m.errs
		panic(err)
	}
}

func (m *Model) Handle(r *types.Request) error {
	var err error

	if err = m.save(r); err != nil {
		return err
	}

	switch r.Metadata.Status {
	case types.DEPOSIT:
		err = m.Scanner.Handle(r)
	case types.SEND:
		err = m.Sender.Handle(r)
	case types.CONFIRM:
		err = m.Monitor.Handle(r)
	case types.EXPIRED:
		return nil
	case types.DONE:
		return nil
	default:
		return ErrUnknownStatus
	}

	r.Metadata.Update()

	if err != nil {
		return err
	}

	return m.Handle(r)
}

var ErrDropMissing = errors.New("drop doesn't exist")

func (m *Model) GetMetadata(a types.Address, d types.Drop, c types.Currency) (*types.Metadata, error) {
	file, err := os.OpenFile(m.path+string(a)+".json", os.O_RDONLY, 0644)
	if err != nil {
		return nil, err
	}

	var data map[types.Currency]map[types.Drop]*types.Metadata

	err = json.NewDecoder(file).Decode(&data)
	if err != nil {
		return nil, err
	}

	if data[c] == nil || data[c][d] == nil {
		return nil, ErrDropMissing
	}

	return data[c][d], nil
}

func (m *Model) load(n string) ([]*types.Request, error) {
	// check that filename is longer than just ".json"
	if len(n) <= 5 {
		return nil, ErrInvalidFilename
	}

	// check that filename is a valid sky address
	addr, err := cipher.DecodeBase58Address(n[:len(n)-5])
	if err != nil {
		return nil, err
	}

	// open file for reading
	file, err := os.OpenFile(m.path+n, os.O_RDONLY, 0644)
	if err != nil {
		return nil, err
	}

	var data map[types.Currency]map[types.Drop]*types.Metadata

	// decode json from file
	err = json.NewDecoder(file).Decode(&data)
	if err != nil {
		return nil, err
	}

	requests := make([]*types.Request, 0)

	for currency, drops := range data {
		for drop, metadata := range drops {
			if metadata.Status == types.DONE {
				continue
			}
			requests = append(requests, &types.Request{
				Address:  types.Address(addr.String()),
				Currency: types.Currency(currency),
				Drop:     types.Drop(drop),
				Metadata: metadata,
			})
		}
	}

	return requests, file.Close()
}

func (m *Model) save(r *types.Request) error {
	var data map[types.Currency]map[types.Drop]*types.Metadata

	// open/create file for reading and writing
	file, err := os.OpenFile(
		m.path+string(r.Address)+".json",
		os.O_CREATE|os.O_RDWR,
		0644,
	)
	if err != nil {
		return err
	}

	// decode file
	err = json.NewDecoder(file).Decode(&data)
	if err != nil && err != io.EOF {
		return err
	}

	// reset
	file.Truncate(0)
	file.Seek(0, 0)
	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")

	// update map
	if data == nil {
		data = map[types.Currency]map[types.Drop]*types.Metadata{
			r.Currency: {r.Drop: r.Metadata},
		}
	} else if data[r.Currency] == nil {
		data[r.Currency] = map[types.Drop]*types.Metadata{
			r.Drop: r.Metadata,
		}
	} else {
		data[r.Currency][r.Drop] = r.Metadata
	}

	// write map to disk
	err = enc.Encode(data)
	if err != nil {
		return err
	}

	err = file.Sync()
	if err != nil {
		return err
	}

	return file.Close()
}
