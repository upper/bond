package bond

import (
	"reflect"

	"upper.io/db"
)

type Store interface {
	db.Collection
	Session() Session
	Save(interface{}) error
	Delete(interface{}) error
	Tx(db.Tx) Store
}

type store struct {
	db.Collection
	session *session
}

// Tx returns a copy of the store that runs in the context of the given
// transaction.
func (s *store) Tx(tx db.Tx) Store {
	return &store{
		Collection: tx.C(s.Collection.Name()),
		session: &session{
			Database: tx,
			stores:   make(map[string]*store),
		},
	}
}

func (s *store) Append(item interface{}) (interface{}, error) {
	if m, ok := item.(HasValidate); ok {
		if err := m.Validate(); err != nil {
			return nil, err
		}
	}

	if m, ok := item.(HasBeforeCreate); ok {
		if err := m.BeforeCreate(); err != nil {
			return nil, err
		}
	}

	id, err := s.Collection.Append(item)
	if err != nil {
		return nil, err
	}

	if m, ok := item.(HasAfterCreate); ok {
		m.AfterCreate()
	}

	return id, nil
}

func (s *store) Find(terms ...interface{}) db.Result {
	result := &result{session: s.session, collection: s.Collection}
	result.args.where = &terms
	return result
}

func (s *store) Save(item interface{}) error {
	pkField, structInfo, err := structMapper.getPrimaryField(item)
	if err != nil {
		return err
	}

	if m, ok := item.(HasValidate); ok {
		if err := m.Validate(); err != nil {
			return err
		}
	}

	id := pkField.Interface()

	if id == structInfo.pkFieldInfo.Zero.Interface() {
		// Create

		if m, ok := item.(HasBeforeCreate); ok {
			if err := m.BeforeCreate(); err != nil {
				return err
			}
		}

		id, err := s.Collection.Append(item)
		if err != nil {
			return err
		}
		pkField.Set(reflect.ValueOf(id))

		if m, ok := item.(HasAfterCreate); ok {
			m.AfterCreate()
		}

	} else {
		// Update

		if m, ok := item.(HasBeforeUpdate); ok {
			if err := m.BeforeUpdate(); err != nil {
				return err
			}
		}

		idKey := structInfo.pkFieldInfo.Name

		err := s.Collection.Find(db.Cond{idKey: id}).Update(item)
		if err != nil {
			return err
		}

		if m, ok := item.(HasAfterUpdate); ok {
			m.AfterUpdate()
		}

	}

	return nil
}

func (s *store) Delete(item interface{}) error {
	pkField, structInfo, err := structMapper.getPrimaryField(item)
	if err != nil {
		return err
	}

	id := pkField.Interface()
	idKey := structInfo.pkFieldInfo.Name

	// Inform when we're deleting an item with no ID value
	if id == structInfo.pkFieldInfo.Zero.Interface() {
		return ErrZeroItemID
	}

	if m, ok := item.(HasBeforeDelete); ok {
		if err := m.BeforeDelete(); err != nil {
			return err
		}
	}

	err = s.Collection.Find(db.Cond{idKey: id}).Remove()
	if err != nil {
		return err
	}

	if m, ok := item.(HasAfterDelete); ok {
		m.AfterDelete()
	}

	return nil
}

func (s *store) Session() Session {
	return s.session
}
