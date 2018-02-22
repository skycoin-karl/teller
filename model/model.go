package model

import (
	"container/list"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"sync"
	"time"

	"github.com/skycoin-karl/teller/types"
	"github.com/skycoin/skycoin/src/cipher"
)

var (
	ErrUnknownStatus   = errors.New("unknown status type")
	ErrInvalidFilename = errors.New("invalid filename in db dir")
)

type Model struct {
	sync.Mutex

	path    string
	errs    chan error
	stop    chan struct{}
	results *list.List
	Scanner types.Service
	Sender  types.Service
	Monitor types.Service
}

func NewModel(path string, scnr, sndr, mntr types.Service) (*Model, error) {
	m := &Model{
		results: list.New().Init(),
		path:    path,
		errs:    make(chan error),
		stop:    make(chan struct{}),
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
			if err == io.EOF {
				continue
			}
			return nil, err
		}

		// inject each request into the proper service
		for _, request := range requests {
			err := m.Add(request)
			if err != nil {
				return nil, err
			}
		}
	}

	return m, nil
}

func (m *Model) Stop() {
	println("stopping scanner")
	m.Scanner.Stop()
	println("stopping sender")
	m.Sender.Stop()
	println("stopping monitor")
	m.Monitor.Stop()

	println("stopping model")
	m.stop <- struct{}{}
}

func (m *Model) Start() {
	go func() {
		for {
			// TODO: tick
			<-time.After(time.Second * 1)

			select {
			case <-m.stop:
				return
			default:
				m.process()
			}
		}
	}()
}

func (m *Model) process() {
	m.Lock()
	defer m.Unlock()

	for e := m.results.Front(); e != nil; e = e.Next() {
		// convert to result promise
		r := e.Value.(chan *types.Result)

		// non-blocking read on each result promise
		select {
		case result := <-r:
			if result.Err != nil {
				m.errs <- result.Err
			} else {
				result.Request.Metadata.Update()
				err := m.save(result.Request)
				if err != nil {
					m.errs <- err
				}
				next := m.Handle(result.Request)
				if next != nil {
					m.results.PushFront(next)
				}
			}
			m.results.Remove(e)
		default:
			continue
		}
	}
}

func (m *Model) logger() {
	for {
		err := <-m.errs
		println(err.Error())
	}
}

func (m *Model) Add(r *types.Request) error {
	m.Lock()
	defer m.Unlock()

	// save to disk
	if err := m.save(r); err != nil {
		return err
	}

	// route to next component
	if res := m.Handle(r); res != nil {
		m.results.PushFront(res)
	}

	return nil
}

func (m *Model) Handle(r *types.Request) chan *types.Result {
	switch r.Metadata.Status {
	case types.DEPOSIT:
		return m.Scanner.Handle(r)
	case types.SEND:
		return m.Sender.Handle(r)
	case types.CONFIRM:
		return m.Monitor.Handle(r)
	case types.EXPIRED:
		fallthrough
	case types.DONE:
		fallthrough
	default:
		return nil
	}
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
