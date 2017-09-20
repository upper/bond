# Bond

Package `bond` (`/bɑnd/`) is a database framework on top of
[upper-db](https://upper.io) which defines a set of hard rules and agreements
on how to work to with SQL databases.

`bond` defines the following concepts:

## Session

* A session represents a database connection pool.
* ...

## Model

* A model is the equivalence between table values and Go values.
* ...

## Store

* A bond store represents the relation between a model and a table, a store
  accepts models and commits operations.
* ...

# Requisites

If you want to use bond, your database design must follow a well-defined set of
rules:

* All models require a table with a primary key (composite primary keys are
  also OK).
* ...

## Example project

Check out [bond-example-project](https://github.com/upper/bond-example-project)
for a project example.

## License

This project is licensed under the terms of the **MIT License**.

> Copyright (c) 2015-present The upper.io/db authors. All rights reserved.
>
> Permission is hereby granted, free of charge, to any person obtaining
> a copy of this software and associated documentation files (the
> "Software"), to deal in the Software without restriction, including
> without limitation the rights to use, copy, modify, merge, publish,
> distribute, sublicense, and/or sell copies of the Software, and to
> permit persons to whom the Software is furnished to do so, subject to
> the following conditions:
>
> The above copyright notice and this permission notice shall be
> included in all copies or substantial portions of the Software.
>
> THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
> EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
> MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
> NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE
> LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION
> OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION
> WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
