DROP TABLE IF EXISTS accounts;

CREATE TABLE accounts (
  id serial primary key,
  name varchar(256),
  disabled boolean,
  created_at timestamp with time zone
);

DROP TABLE IF EXISTS users;

CREATE TABLE users (
  id serial primary key,
  account_id integer,
  username varchar(256) UNIQUE
);

DROP TABLE IF EXISTS logs;

CREATE TABLE logs (
  id serial primary key,
	message VARCHAR
);
