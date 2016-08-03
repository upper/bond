package bond

import (
	"reflect"

	"upper.io/db.v2"
)

type Store interface {
	db.Collection

	Session() Session
	Save(interface{}) error
	Delete(interface{}) error
	With(sess Session) Store
}

type store struct {
	db.Collection

	session Session
}

// With returns a copy of the store that runs in the context of the given
// transaction.
func (s *store) With(sess Session) Store {
	return &store{
		Collection: sess.(*session).Collection(s.Collection.Name()),
		session:    sess,
	}
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
		if err = m.Validate(); err != nil {
			return err
		}
	}

	id := pkField.Interface()

	if id == structInfo.pkFieldInfo.Zero.Interface() {
		return s.Create(item)
	} else {
		return s.Update(item)
	}

	return nil
}

func (s *store) Create(item interface{}) error {
	pkField, _, err := structMapper.getPrimaryField(item)
	if err != nil {
		return err
	}

	if m, ok := item.(HasBeforeCreate); ok {
		if err = m.BeforeCreate(s.session); err != nil {
			return err
		}
	}

	id, err := s.Collection.Insert(item)
	if err != nil {
		return err
	}
	pkField.Set(reflect.ValueOf(id))

	if m, ok := item.(HasAfterCreate); ok {
		if err = m.AfterCreate(s.session); err != nil {
			return err
		}
	}
	return nil
}

func (s *store) Update(item interface{}) error {
	pkField, structInfo, err := structMapper.getPrimaryField(item)
	if err != nil {
		return err
	}
	id := pkField.Interface()

	if m, ok := item.(HasBeforeUpdate); ok {
		if err = m.BeforeUpdate(s.session); err != nil {
			return err
		}
	}

	idKey := structInfo.pkFieldInfo.Name

	if err = s.Collection.Find(db.Cond{idKey: id}).Update(item); err != nil {
		return err
	}

	if m, ok := item.(HasAfterUpdate); ok {
		if err = m.AfterUpdate(s.session); err != nil {
			return err
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
		if err = m.BeforeDelete(s.session); err != nil {
			return err
		}
	}

	if err = s.Collection.Find(db.Cond{idKey: id}).Delete(); err != nil {
		return err
	}

	if m, ok := item.(HasAfterDelete); ok {
		if err = m.AfterDelete(s.session); err != nil {
			return err
		}
	}

	return nil
}

func (s *store) Session() Session {
	return s.session
}
