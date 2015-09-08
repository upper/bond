package bond

import (
	"reflect"
	"sync"

	"upper.io/db"
)

type Session interface {
	db.Tx
	Store(interface{}) Store
	Find(...interface{}) db.Result
	Save(Model) error
	Delete(Model) error
	NewTransaction() (Session, error)
}

type session struct {
	db.Database
	stores     map[string]*store
	storesLock sync.Mutex
}

func Open(adapter string, url db.ConnectionURL) (Session, error) {
	conn, err := db.Open(adapter, url)
	if err != nil {
		return nil, err
	}

	sess := &session{
		Database: conn,
		stores:   make(map[string]*store),
	}

	return sess, nil
}

// NewTransaction creates and returns a session that runs within a transaction
// block.
func (s *session) NewTransaction() (Session, error) {

	tx, err := s.Database.Transaction()
	if err != nil {
		return nil, err
	}

	sess := &session{
		Database: tx,
		stores:   make(map[string]*store),
	}

	return sess, nil
}

// Commit commits the current transaction.
func (s *session) Commit() error {
	if tx, ok := s.Database.(db.Tx); ok {
		return tx.Commit()
	}
	return ErrMissingTransaction
}

// Rollback discards the current transaction.
func (s *session) Rollback() error {
	if tx, ok := s.Database.(db.Tx); ok {
		return tx.Rollback()
	}
	return ErrMissingTransaction
}

func (s *session) Store(item interface{}) Store {
	store := s.getStore(item)
	return store
}

func (s *session) Find(terms ...interface{}) db.Result {
	result := &result{session: s}
	if len(terms) > 0 {
		result.args.where = &terms
	}
	return result
}

func (s *session) Save(item Model) error {
	store := s.getStore(item)
	return store.Save(item)
}

func (s *session) Delete(item Model) error {
	store := s.getStore(item)
	return store.Delete(item)
}

func (s *session) getStore(item interface{}) *store {
	var colName string

	if str, ok := item.(string); ok {
		colName = str
	} else if m, ok := item.(Model); ok {
		colName = m.CollectionName()
	}

	if colName == "" {
		itemv := reflect.ValueOf(item)
		if itemv.Kind() == reflect.Ptr {
			itemv = reflect.Indirect(itemv)
		}
		item = itemv.Interface()
		if m, ok := item.(Model); ok {
			colName = m.CollectionName()
		}
	}

	if colName == "" {
		panic(ErrUnknownCollection)
	}

	s.storesLock.Lock()
	defer s.storesLock.Unlock()

	if store, ok := s.stores[colName]; ok {
		return store
	}

	store := &store{Collection: s.Database.C(colName), session: s}
	s.stores[colName] = store
	return store
}
