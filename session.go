package bond

import (
	"fmt"
	"reflect"
	"sync"

	"upper.io/db"
)

type Session interface {
	db.Database
	Store(interface{}) Store
	Find(...interface{}) db.Result
	Save(Model) error
	Delete(Model) error
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

	col, err := s.Database.Collection(colName)
	if err != nil {
		panic(fmt.Errorf("%v: %v", colName, err))
	}

	store := &store{Collection: col, session: s}
	s.stores[colName] = store
	return store
}
