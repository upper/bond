package bond

import (
	"database/sql"
	"fmt"
	"reflect"
	"sync"

	"upper.io/db.v2"
	"upper.io/db.v2/lib/sqlbuilder"
)

// SQLBackend represents both *sql.Tx and *sql.DB.
type SQLBackend interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
	Prepare(query string) (*sql.Stmt, error)
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row
}

type Session interface {
	sqlbuilder.Backend

	Store(interface{}) Store
	Find(...interface{}) db.Result
	Save(Model) error
	Delete(Model) error

	SessionTx(func(tx Session) error) error
}

type session struct {
	sqlbuilder.Backend

	stores     map[string]*store
	storesLock sync.Mutex
}

// Open connects to a database.
func Open(adapter string, url db.ConnectionURL) (Session, error) {
	conn, err := sqlbuilder.Open(adapter, url)
	if err != nil {
		return nil, err
	}
	return New(conn), nil
}

// New returns a new session.
func New(conn sqlbuilder.Backend) Session {
	return &session{Backend: conn, stores: make(map[string]*store)}
}

// Bind binds to an existent database session. Possible backend values are:
// *sql.Tx or *sql.DB.
func Bind(adapter string, backend SQLBackend) (Session, error) {
	var conn sqlbuilder.Backend

	switch t := backend.(type) {
	case *sql.Tx:
		var err error
		conn, err = sqlbuilder.NewTx(adapter, t)
		if err != nil {
			return nil, err
		}
	case *sql.DB:
		var err error
		conn, err = sqlbuilder.New(adapter, t)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("Unknown backend type: %T", t)
	}
	return &session{Backend: conn, stores: make(map[string]*store)}, nil
}

func (s *session) SessionTx(fn func(sess Session) error) error {
	txFn := func(sess sqlbuilder.Tx) error {
		return fn(&session{
			Backend: sess,
			stores:  make(map[string]*store),
		})
	}

	switch t := s.Backend.(type) {
	case sqlbuilder.Database:
		return t.Tx(txFn)
	case sqlbuilder.Tx:
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

	store := &store{Collection: s.Collection(colName), session: s}
	s.stores[colName] = store
	return store
}
