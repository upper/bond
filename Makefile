TEST_HOST ?= 127.0.0.1
TEST_PORT ?= 5432
BOND_DB   ?= bond_test

all: test

build:
	@go build

test: resetdb
	UPPERIO_DB_DEBUG=1 go test -v ./...

resetdb:
	psql -Upostgres -h$(TEST_HOST) -p$(TEST_PORT) -c "DROP DATABASE IF EXISTS $(BOND_DB)" && \
	psql -Upostgres -h$(TEST_HOST) -p$(TEST_PORT) -c "DROP ROLE IF EXISTS bond_user" && \
	psql -Upostgres -h$(TEST_HOST) -p$(TEST_PORT) -c "CREATE USER bond_user" && \
	psql -Upostgres -h$(TEST_HOST) -p$(TEST_PORT) -c "CREATE DATABASE $(BOND_DB) ENCODING 'UTF-8' LC_COLLATE='en_US.UTF-8' LC_CTYPE='en_US.UTF-8' TEMPLATE template0" && \
	psql -Upostgres -h$(TEST_HOST) -p$(TEST_PORT) -c "GRANT ALL PRIVILEGES ON DATABASE $(BOND_DB) TO bond_user" && \
	psql -Ubond_user -h$(TEST_HOST) -p$(TEST_PORT) $(BOND_DB) < test_schema.sql
