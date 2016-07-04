package bond

import (
	"reflect"
	"sync"

	"upper.io/db.v2"
	"upper.io/db.v2/postgresql" // TODO: Figure out how to remove this and make it generic.
)

type Session interface {
	postgresql.Tx
	Store(interface{}) Store
	Find(...interface{}) db.Result
	Save(Model) error
	Delete(Model) error
	Tx() (Session, error)
	ContinueTransaction() (Session, error, bool)
}

type session struct {
	postgresql.Database
	stores     map[string]*store
	storesLock sync.Mutex
}

// Open connects to a database.
func Open(adapter string, url db.ConnectionURL) (Session, error) {
	conn, err := postgresql.Open(url)
	if err != nil {
		return nil, err
	}

	return &session{Database: conn, stores: make(map[string]*store)}, nil
}

// Tx creates and returns a session that runs within a transaction
// block. It will fail if called inside another transaction
func (s *session) Tx() (Session, error) {
	tx, err := s.Database.NewTransaction()
	if err != nil {
		return nil, err
	}

	sess := &session{
		Database: tx,
		stores:   make(map[string]*store),
	}

	return sess, nil
}

// ContinueTransaction creates and returns a session that runs within a
// transaction block. If called within another transaction block it will reuse
// the transaction in progress, if not it will start a new transaction.
// The 3rd returned value indicates if a session was continued (true) or not
func (s *session) ContinueTransaction() (Session, error, bool) {
	// check if called within a transaction
	tx, inTransaction := s.Database.(postgresql.Tx)

	// if not start a new one
	if !inTransaction {
		var err error
		tx, err = s.Database.NewTransaction()
		if err != nil {
			return nil, err, false
		}
	}

	sess := &session{
		Database: tx,
		stores:   make(map[string]*store),
	}

	return sess, nil, inTransaction
}

// Commit commits the current transaction.
func (s *session) Commit() error {
	if tx, ok := s.Database.(postgresql.Tx); ok {
		return tx.Commit()
	}
	return ErrMissingTransaction
}

// Rollback discards the current transaction.
func (s *session) Rollback() error {
	if tx, ok := s.Database.(postgresql.Tx); ok {
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

	store := &store{Collection: s.Database.Collection(colName), session: s}
	s.stores[colName] = store
	return store
}
