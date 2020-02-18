PROJECT      ?= bond

DB_HOST      ?= 127.0.0.1
DB_PORT      ?= 5432
DB_USER      ?= postgres
DB_PASSWORD  ?= postgr3s

BOND_DB         ?= bond_test
BOND_USER       ?= bond_user
BOND_PASSWORD   ?= bond_password

TEST_FLAGS ?=

export POSTGRES_VERSION

export DB_HOST
export DB_NAME
export DB_PASSWORD
export DB_PORT
export DB_USERNAME

export TEST_FLAGS

all: test

test: db-reset
	UPPERIO_DB_DEBUG=1 go test -v ./... $(TEST_FLAGS)

db-reset:
	export PGPASSWORD="$(DB_PASSWORD)" && \
	psql -U$(DB_USER) -h$(DB_HOST) -p$(DB_PORT) -c "DROP DATABASE IF EXISTS $(BOND_DB)" && \
	psql -U$(DB_USER) -h$(DB_HOST) -p$(DB_PORT) -c "DROP ROLE IF EXISTS $(BOND_USER)" && \
	psql -U$(DB_USER) -h$(DB_HOST) -p$(DB_PORT) -c "CREATE USER $(BOND_USER) WITH PASSWORD '$(BOND_PASSWORD)'" && \
	psql -U$(DB_USER) -h$(DB_HOST) -p$(DB_PORT) -c "CREATE DATABASE $(BOND_DB) ENCODING 'UTF-8' LC_COLLATE='en_US.UTF-8' LC_CTYPE='en_US.UTF-8' TEMPLATE template0" && \
	psql -U$(DB_USER) -h$(DB_HOST) -p$(DB_PORT) -c "GRANT ALL PRIVILEGES ON DATABASE $(BOND_DB) TO $(BOND_USER)"
	export PGPASSWORD="$(BOND_PASSWORD)" && \
	psql -U$(BOND_USER) -h$(DB_HOST) -p$(DB_PORT) $(BOND_DB) < test_schema.sql

server-up: server-down
	docker-compose -p $(PROJECT) up -d && \
	sleep 10

server-down:
	docker-compose -p $(PROJECT) down
