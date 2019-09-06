package bond

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"sync"

	"github.com/pkg/errors"
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
	db.Database
	sqlbuilder.SQLBuilder

	SetTxOptions(sql.TxOptions)
	TxOptions() *sql.TxOptions
}

type Session interface {
	Backend

	Conn() sqlbuilder.Database

	Store(collectionName string) Store
	ResolveStore(interface{}) Store

	Save(Model) error
	Delete(Model) error

	WithContext(context.Context) Session
	Context() context.Context

	SessionTx(context.Context, func(tx Session) error) error
	NewTx(context.Context) (sqlbuilder.Tx, error)
	NewSessionTx(context.Context) (Session, error)

	TxCommit() error
	TxRollback() error
}

type session struct {
	Backend

	stores map[string]*store
	mu     sync.Mutex
}

// Open connects to a database.
func Open(adapter string, url db.ConnectionURL) (Session, error) {
	conn, err := sqlbuilder.Open(adapter, url)
	if err != nil {
		return nil, err
	}

	sess := New(conn)
	return sess, nil
}

// New returns a new session.
func New(conn Backend) Session {
	return &session{Backend: conn, stores: make(map[string]*store)}
}

func (s *session) Conn() sqlbuilder.Database {
	return s.Backend.(sqlbuilder.Database)
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

func (s *session) NewTx(ctx context.Context) (sqlbuilder.Tx, error) {
	return s.Conn().NewTx(ctx)
}

func (s *session) NewSessionTx(ctx context.Context) (Session, error) {
	tx, err := s.NewTx(ctx)
	if err != nil {
		return nil, err
	}
	return &session{
		Backend: tx,
		stores:  make(map[string]*store),
	}, nil
}

func (s *session) TxCommit() error {
	tx, ok := s.Backend.(sqlbuilder.Tx)
	if !ok {
		return errors.Errorf("bond: session is not a tx")
	}
	defer tx.Close()
	return tx.Commit()
}

func (s *session) TxRollback() error {
	tx, ok := s.Backend.(sqlbuilder.Tx)
	if !ok {
		return errors.Errorf("bond: session is not a tx")
	}
	defer tx.Close()
	return tx.Rollback()
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
			if rErr := t.Rollback(); rErr != nil {
				return errors.Wrap(err, rErr.Error())
			}
			return err
		}
		return t.Commit()
	}

	return errors.New("Missing backend, forgot to use bond.New?")
}

func (s *session) getStore(item interface{}) Store {
	if c, ok := item.(HasStore); ok {
		return c.Store(s)
	}
	if c, ok := item.(HasCollectionName); ok {
		return s.Store(c.CollectionName())
	}
	panic("reached")
}

func (s *session) Save(item Model) error {
	if item == nil {
		return ErrExpectingNonNilModel
	}
	return s.getStore(item).Save(item)
}

func (s *session) Delete(item Model) error {
	if item == nil {
		return ErrExpectingNonNilModel
	}
	return s.getStore(item).Delete(item)
}

func (s *session) Store(collectionName string) Store {
	if collectionName == "" {
		return &store{session: s}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if store, ok := s.stores[collectionName]; ok {
		return store
	}

	store := &store{
		Collection: s.Collection(collectionName),
		session:    s,
	}
	s.stores[collectionName] = store
	return store
}

func (s *session) ResolveStore(item interface{}) Store {
	var colName string

	switch t := item.(type) {
	case string:
		colName = t
	case func(sess Session) db.Collection:
		colName = t(s).Name()
	case Store:
		return t
	case db.Collection:
		colName = t.Name()
	case Model:
		colName = t.Store(s).Name()
	default:
		itemv := reflect.ValueOf(item)
		if itemv.Kind() == reflect.Ptr {
			itemv = reflect.Indirect(itemv)
		}
		item = itemv.Interface()
		if m, ok := item.(Model); ok {
			colName = m.Store(s).Name()
		}
	}

	return s.Store(colName)
}
