package bond

import "upper.io/db"

type result struct {
	session    Session
	collection db.Collection
	query      db.Result

	args struct {
		where  *[]interface{}
		limit  *uint
		skip   *uint
		sort   *[]interface{}
		fields *[]interface{}
		group  *[]interface{}
	}
}

// TODO: would be nice if db.Result had a Query() (interface{}, error)
// method that would return the raw query that is sent to the driver.
// Useful for debugging..

func (r *result) Limit(n uint) db.Result {
	r.args.limit = &n
	return r
}

func (r *result) Skip(n uint) db.Result {
	r.args.skip = &n
	return r
}

func (r *result) Sort(fields ...interface{}) db.Result {
	r.args.sort = &fields
	return r
}

func (r *result) Select(fields ...interface{}) db.Result {
	r.args.fields = &fields
	return r
}

func (r *result) Where(terms ...interface{}) db.Result {
	r.args.where = &terms
	return r
}

func (r *result) Group(fields ...interface{}) db.Result {
	r.args.group = &fields
	return r
}

func (r *result) One(dst interface{}) error {
	var col db.Collection

	if r.collection != nil {
		col = r.collection
	} else {
		col = r.getCollection(dst)
	}

	res, err := r.buildQuery(col)
	if err != nil {
		return err
	}
	r.query = res

	return res.One(dst)
}

func (r *result) All(dst interface{}) error {
	var col db.Collection

	if r.collection != nil {
		col = r.collection
	} else {
		col = r.getCollection(dst)
	}

	res, err := r.buildQuery(col)
	if err != nil {
		return err
	}
	r.query = res

	return res.All(dst)
}

func (r *result) Next(dst interface{}) error {
	var col db.Collection

	if r.collection != nil {
		col = r.collection
	} else {
		col = r.getCollection(dst)
	}

	res, err := r.buildQuery(col)
	if err != nil {
		return err
	}
	r.query = res

	return res.Next(dst)
}

func (r *result) Update(values interface{}) error {
	if r.collection == nil {
		return ErrUnknownCollection
	}

	res, err := r.buildQuery(r.collection)
	if err != nil {
		return err
	}
	r.query = res

	return res.Update(values)
}

func (r *result) Remove() error {
	if r.collection == nil {
		return ErrUnknownCollection
	}

	res, err := r.buildQuery(r.collection)
	if err != nil {
		return err
	}
	r.query = res

	return res.Remove()
}

func (r *result) Count() (uint64, error) {
	if r.collection == nil {
		return 0, ErrUnknownCollection
	}

	res, err := r.buildQuery(r.collection)
	if err != nil {
		return 0, err
	}
	r.query = res

	return res.Count()
}

func (r *result) Close() error {
	if r.query == nil {
		return ErrInvalidQuery
	}
	return r.query.Close()
}

func (r *result) getCollection(dst interface{}) db.Collection {
	if r.collection != nil {
		return r.collection
	}
	r.collection = r.session.Store(dst)
	return r.collection
}

func (r *result) buildQuery(col db.Collection) (db.Result, error) {
	if r.args.where == nil {
		return nil, ErrInvalidQuery
	}
	res := col.Find((*r.args.where)...)

	if r.args.limit != nil {
		res = res.Limit(*r.args.limit)
	}
	if r.args.skip != nil {
		res = res.Skip(*r.args.skip)
	}
	if r.args.sort != nil {
		res = res.Sort(*r.args.sort)
	}
	if r.args.fields != nil {
		res = res.Select(*r.args.fields)
	}
	if r.args.group != nil {
		res = res.Group(*r.args.group)
	}

	return res, nil
}
