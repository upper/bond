package bond_test

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"log"
	"testing"
	"upper.io/bond"
	"upper.io/db.v3/postgresql"
)

type UserModel struct {
	ID        int64  `db:"id,omitempty,pk"`
	AccountID int64  `db:"account_id"`
	Username  string `db:"username"`
}

var _ bond.Model = &UserModel{}

func (um *UserModel) Save(sess bond.Session) error {
	if um.ID > 0 {
		return um.Store(sess).UpdateReturning(um)
	}
	return um.Store(sess).InsertReturning(um)
}

func (um *UserModel) Store(sess bond.Session) bond.Store {
	return repo.Users(sess)
}

func (um *UserModel) BeforeCreate(sess bond.Session) error {
	log.Printf("BeforeCreate")
	return nil
}

func (um *UserModel) AfterCreate(sess bond.Session) error {
	log.Printf("AfterCreate")
	return nil
}

func (um *UserModel) SetAccountID(sess bond.Session) error {
	um.AccountID = 5
	return nil
}

type Repo struct {
}

func (r *Repo) Users(sess bond.Session) bond.Store {
	return sess.Store("users")
}

var repo = &Repo{}

func settings() postgresql.ConnectionURL {
	return postgresql.ConnectionURL{
		Host:     fmt.Sprintf("%s:%s", pickDefault("DB_HOST", "127.0.0.1"), pickDefault("DB_PORT", "5432")),
		Database: pickDefault("BOND_DB", "bond_test"),
		User:     pickDefault("BOND_USER", "bond_user"),
		Password: pickDefault("BOND_PASSWORD", "bond_password"),
	}
}

func dbUp() bond.Session {
	conn, err := postgresql.Open(settings())
	if err != nil {
		panic(err)
	}

	sess := bond.New(conn)

	cols, _ := sess.Collections()
	for _, c := range cols {
		sess.Collection(c).Truncate()
	}

	return sess
}

func TestRepo(t *testing.T) {
	sess := dbUp()
	defer sess.Close()

	user := &UserModel{Username: "Leia"}

	err := sess.Save(user)
	assert.NoError(t, err)

	assert.NotZero(t, user.ID)

	err = user.SetAccountID(sess)
	assert.NoError(t, err)

	user.Username = "Scandal"
	err = sess.Save(user)
	assert.NoError(t, err)

	assert.NotZero(t, user.ID)

	err = sess.Delete(user)
	assert.NoError(t, err)

	user = &UserModel{Username: "Leia"}

	err = sess.Save(user)
	assert.NoError(t, err)

	assert.NotZero(t, user.ID)

	var user2 UserModel

	err = repo.Users(sess).Find(user.ID).One(&user2)
	assert.NoError(t, err)
	assert.NotZero(t, user2.ID)

	err = sess.SessionTx(nil, func(tx bond.Session) error {
		var user UserModel
		err = repo.Users(tx).Find().Limit(1).One(&user)
		assert.NoError(t, err)
		assert.NotZero(t, user.ID)
		return nil
	})

	assert.NoError(t, err)
}
