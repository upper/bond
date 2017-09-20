package bond

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"sync"

	"upper.io/db.v3"
	"upper.io/db.v3/lib/sqlbuilder"
)

type txWithContext interface {
	WithContext(context.Context) sqlbuilder.Tx
}

type databaseWithContext interface {
	WithContext(context.Context) sqlbuilder.Database
}

type hasContext interface {
	Context() context.Context
}

// SQLBackend represents a type that can execute SQL queries.
type SQLBackend interface {
	Exec(string, ...interface{}) (sql.Result, error)
	Prepare(string) (*sql.Stmt, error)
	Query(string, ...interface{}) (*sql.Rows, error)
	QueryRow(string, ...interface{}) *sql.Row
}

type Backend interface {
	sqlbuilder.SQLBuilder
	db.Database

	SetTxOptions(sql.TxOptions)
	TxOptions() *sql.TxOptions
}

type Session interface {
	Backend

	Store(interface{}) Store

	Save(Model) error
	Delete(Model) error

	WithContext(context.Context) Session
	Context() context.Context

	SessionTx(context.Context, func(tx Session) error) error
}

type session struct {
	Backend

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
func New(conn Backend) Session {
	return &session{Backend: conn, stores: make(map[string]*store)}
}

func (s *session) WithContext(ctx context.Context) Session {
	var backendCtx Backend
	switch t := s.Backend.(type) {
	case databaseWithContext:
		backendCtx = t.WithContext(ctx)
	case txWithContext:
		backendCtx = t.WithContext(ctx)
	default:
		panic("Bad session")
	}

	return &session{
		Backend: backendCtx,
		stores:  make(map[string]*store),
	}
}

func (s *session) Context() context.Context {
	return s.Backend.(hasContext).Context()
}

// Bind binds to an existent database session. Possible backend values are:
// *sql.Tx or *sql.DB.
func Bind(adapter string, backend SQLBackend) (Session, error) {
	var conn Backend

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

func (s *session) SessionTx(ctx context.Context, fn func(sess Session) error) error {
	txFn := func(sess sqlbuilder.Tx) error {
		return fn(&session{
			Backend: sess,
			stores:  make(map[string]*store),
		})
	}

	switch t := s.Backend.(type) {
	case sqlbuilder.Database:
		return t.Tx(ctx, txFn)
	case sqlbuilder.Tx:
		defer t.Close()
		err := txFn(t)
		if err != nil {
			return t.Rollback()
		}
		return t.Commit()
	}

	return errors.New("Missing backend, forgot to use bond.New?")
}

func (s *session) Save(item Model) error {
	if item == nil {
		return ErrExpectingNonNilModel
	}
	return s.Store(item.CollectionName()).Save(item)
}

func (s *session) Delete(item Model) error {
	if item == nil {
		return ErrExpectingNonNilModel
	}
	return s.Store(item.CollectionName()).Delete(item)
}

func (s *session) Store(item interface{}) Store {
	var colName string

	switch t := item.(type) {
	case string:
		colName = t
	case Model:
		colName = t.CollectionName()
	default:
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
		return &store{session: s}
	}

	s.storesLock.Lock()
	defer s.storesLock.Unlock()

	if store, ok := s.stores[colName]; ok {
		return store
	}

	store := &store{
		Collection: s.Collection(colName),
		session:    s,
	}
	s.stores[colName] = store
	return store
}
