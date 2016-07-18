package bond

import (
	"database/sql"
	"fmt"
	"reflect"
	"sync"

	"upper.io/db.v2"
)

// SQLDatabase represents a normal session and transaction session.
type SQLDatabase interface {
	db.Database
	db.SQLBuilder
}

type Session interface {
	SQLDatabase

	Store(interface{}) Store
	Find(...interface{}) db.Result
	Save(Model) error
	Delete(Model) error

	SessionTx(func(tx Session) error) error
}

type session struct {
	SQLDatabase
	stores     map[string]*store
	storesLock sync.Mutex
}

// Open connects to a database.
func Open(adapter string, url db.ConnectionURL) (Session, error) {
	conn, err := db.SQLAdapter(adapter).Open(url)
	if err != nil {
		return nil, err
	}
	return New(adapter, conn)
}

// New returns a new session.
func New(adapter string, backend interface{}) (Session, error) {
	var conn SQLDatabase

	switch t := backend.(type) {
	case SQLDatabase:
		conn = t
	case *sql.Tx:
		var err error
		conn, err = db.SQLAdapter(adapter).NewTx(t)
		if err != nil {
			return nil, err
		}
	case *sql.DB:
		var err error
		conn, err = db.SQLAdapter(adapter).New(t)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("Unknown backend type: %T", t)
	}
	return &session{SQLDatabase: conn, stores: make(map[string]*store)}, nil
}

func (s *session) SessionTx(fn func(sess Session) error) error {
	txFn := func(sess db.SQLTx) error {
		return fn(&session{
			SQLDatabase: sess,
			stores:      make(map[string]*store),
		})
	}

	switch t := s.SQLDatabase.(type) {
	case db.SQLDatabase:
		return t.Tx(txFn)
	case db.SQLTx:
		defer t.Close()
		err := txFn(t)
		if err != nil {
			return t.Rollback()
		}
		return t.Commit()
	}
	panic("reached")
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

	store := &store{Collection: s.SQLDatabase.Collection(colName), session: s}
	s.stores[colName] = store
	return store
}
