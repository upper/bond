package bond

import "upper.io/db.v2"

type result struct {
	session    Session
	collection db.Collection
	query      db.Result
	lastErr    error

	args struct {
		where  *[]interface{}
		limit  *int
		skip   *int
		sort   *[]interface{}
		fields *[]interface{}
		group  *[]interface{}
	}
}

// TODO: would be nice if db.Result had a Query() (interface{}, error)
// method that would return the raw query that is sent to the driver.
// Useful for debugging..

func (r *result) String() string {
	return r.query.String()
}

func (r *result) Err() error {
	return r.lastErr
}

func (r *result) Limit(n int) db.Result {
	r.args.limit = &n
	return r
}

func (r *result) Offset(n int) db.Result {
	r.args.skip = &n
	return r
}

func (r *result) OrderBy(fields ...interface{}) db.Result {
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

func (r *result) And(terms ...interface{}) db.Result {
	if r.args.where == nil {
		return r.Where(terms...)
	}
	*r.args.where = append(*r.args.where, terms...)
	return r
}

func (r *result) Group(fields ...interface{}) db.Result {
	r.args.group = &fields
	return r
}

func (r *result) One(dst interface{}) error {
	if r.collection == nil {
		r.collection = r.getCollection(dst)
	}
	col := r.collection

	res, err := r.buildQuery(col)
	if err != nil {
		return err
	}
	r.query = res

	return res.One(dst)
}

func (r *result) All(dst interface{}) error {
	if r.collection == nil {
		r.collection = r.getCollection(dst)
	}
	col := r.collection

	res, err := r.buildQuery(col)
	if err != nil {
		return err
	}
	r.query = res

	return res.All(dst)
}

func (r *result) Next(dst interface{}) bool {
	if r.collection == nil {
		r.collection = r.getCollection(dst)
	}
	col := r.collection

	if r.query == nil {
		res, err := r.buildQuery(col)
		if err != nil {
			r.lastErr = err
			return false
		}
		r.query = res
	}

	if !r.query.Next(dst) {
		r.lastErr = r.query.Err()
		return false
	}

	return true
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

func (r *result) Delete() error {
	if r.collection == nil {
		return ErrUnknownCollection
	}

	res, err := r.buildQuery(r.collection)
	if err != nil {
		return err
	}
	r.query = res

	return res.Delete()
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
	return r.session.Store(dst)
}

func (r *result) buildQuery(col db.Collection) (db.Result, error) {
	var res db.Result

	if r.args.where == nil {
		res = col.Find(db.Cond{})
	} else {
		res = col.Find((*r.args.where)...)
	}
	if r.args.limit != nil {
		res = res.Limit(*r.args.limit)
	}
	if r.args.skip != nil {
		res = res.Offset(*r.args.skip)
	}
	if r.args.sort != nil {
		res = res.OrderBy((*r.args.sort)...)
	}
	if r.args.fields != nil {
		res = res.Select((*r.args.fields)...)
	}
	if r.args.group != nil {
		res = res.Group((*r.args.group)...)
	}

	return res, nil
}
