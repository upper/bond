package bond

import "errors"

var (
	ErrUnknownCollection = errors.New("unknown collection")
	ErrInvalidQuery      = errors.New("invalid query")
	ErrZeroItemID        = errors.New("item id is empty")
)

type Model interface {
	HasCollectionName
}

type HasCollectionName interface {
	CollectionName() string
}

type HasValidate interface {
	Validate() error
}

type HasBeforeCreate interface {
	BeforeCreate() error
}

type HasAfterCreate interface {
	AfterCreate()
}

type HasBeforeUpdate interface {
	BeforeUpdate() error
}

type HasAfterUpdate interface {
	AfterUpdate()
}

type HasBeforeDelete interface {
	BeforeDelete() error
}

type HasAfterDelete interface {
	AfterDelete()
}
