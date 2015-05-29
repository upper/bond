BOND_DB = "bond_test"

all: test

build:
	@go build

test:
	@go test -v ./...

resetdb:
	psql -Upostgres <<< "DROP DATABASE $(BOND_DB)"
	psql -Upostgres <<< "CREATE DATABASE $(BOND_DB) ENCODING 'UTF-8' LC_COLLATE='en_US.UTF-8' LC_CTYPE='en_US.UTF-8' TEMPLATE template0;"
	psql -Upostgres $(BOND_DB) < test_schema.sql
