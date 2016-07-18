package bond

import (
	"database/sql"
	"fmt"
	"reflect"
	"sync"

	"upper.io/db.v2"
)

// SQLSession represents both db.SQLDatabase and db.SQLTx interfaces.
type SQLSession interface {
	db.Database
	db.SQLBuilder
}

// SQLBackend represents both *sql.Tx and *sql.DB.
type SQLBackend interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
	Prepare(query string) (*sql.Stmt, error)
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row
}

type Session interface {
	SQLSession

	Store(interface{}) Store
	Find(...interface{}) db.Result
	Save(Model) error
	Delete(Model) error

	SessionTx(func(tx Session) error) error
}

type session struct {
	SQLSession
	stores     map[string]*store
	storesLock sync.Mutex
}

// Open connects to a database.
func Open(adapter string, url db.ConnectionURL) (Session, error) {
	conn, err := db.SQLAdapter(adapter).Open(url)
	if err != nil {
		return nil, err
	}
	return New(conn), nil
}

// New returns a new session.
func New(conn SQLSession) Session {
	return &session{SQLSession: conn, stores: make(map[string]*store)}
}

// Bind binds to an existent database session. Possible backend values are:
// *sql.Tx or *sql.DB.
func Bind(adapter string, backend SQLBackend) (Session, error) {
	var conn SQLSession

	switch t := backend.(type) {
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
	return &session{SQLSession: conn, stores: make(map[string]*store)}, nil
}

func (s *session) SessionTx(fn func(sess Session) error) error {
	txFn := func(sess db.SQLTx) error {
		return fn(&session{
			SQLSession: sess,
			stores:     make(map[string]*store),
		})
	}

	switch t := s.SQLSession.(type) {
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

	store := &store{Collection: s.SQLSession.Collection(colName), session: s}
	s.stores[colName] = store
	return store
}
