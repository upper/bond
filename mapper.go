package bond

import (
	"errors"
	"reflect"
	"sync"

	"upper.io/db.v2/lib/reflectx"
)

var (
	structMapper = newMapper()

	ErrExpectingStruct   = errors.New("item must be a struct")
	ErrMissingPrimaryKey = errors.New("struct fields missing a primary key option (,pk)")
)

type mapper struct {
	*reflectx.Mapper
	cache     map[reflect.Type]*structInfo
	cacheLock sync.Mutex
}

type structInfo struct {
	*reflectx.StructMap
	pkFieldInfo *reflectx.FieldInfo
}

func newMapper() *mapper {
	return &mapper{
		Mapper: reflectx.NewMapper("db"),
		cache:  map[reflect.Type]*structInfo{},
	}
}

func (m *mapper) getStructInfo(item interface{}) (*structInfo, reflect.Value, error) {
	itemv := reflect.ValueOf(item)
	if itemv.Kind() == reflect.Ptr {
		itemv = reflect.Indirect(itemv)
	}

	t := itemv.Type()
	if t.Kind() != reflect.Struct {
		return nil, itemv, ErrExpectingStruct
	}

	m.cacheLock.Lock()
	sinfo, ok := m.cache[t]
	m.cacheLock.Unlock()

	if ok {
		return sinfo, itemv, nil
	}

	sinfo = &structInfo{StructMap: m.TypeMap(t)}

	for _, f := range sinfo.Index {
		if _, ok := f.Options["pk"]; ok {
			sinfo.pkFieldInfo = f
			break
		}
	}

	m.cacheLock.Lock()
	m.cache[t] = sinfo
	m.cacheLock.Unlock()

	return sinfo, itemv, nil
}

func (m *mapper) getPrimaryField(item interface{}) (reflect.Value, *structInfo, error) {
	if item == nil {
		return reflect.Value{}, nil, errors.New("Expecting a non nil value")
	}
	sinfo, itemv, err := m.getStructInfo(item)
	if err != nil {
		return reflect.Value{}, sinfo, err
	}
	if sinfo.pkFieldInfo == nil {
		return reflect.Value{}, sinfo, ErrMissingPrimaryKey
	}
	return itemv.FieldByIndex(sinfo.pkFieldInfo.Index), sinfo, nil
}
