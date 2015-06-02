package bond_test

import (
	"log"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"upper.io/bond"
	"upper.io/db"
	_ "upper.io/db/postgresql"
)

var (
	DB *database
)

type database struct {
	bond.Session

	Account AccountStore `collection:"accounts"`
	User    UserStore    `collection:"users"`
}

type Account struct {
	ID        int64     `db:"id,omitempty,pk"`
	Name      string    `db:"name"`
	Disabled  bool      `db:"disabled"`
	CreatedAt time.Time `db:"created_at"`
}

func (a *Account) CollectionName() string {
	return DB.Account.Name()
}

func (a *Account) BeforeDelete() error {
	// TODO: we should have flags on the object that we set here..
	// and easily reset.. for testing
	log.Println("beforedelete()..")
	return nil
}

type User struct {
	ID        int64  `db:"id,omitempty,pk"`
	AccountID int64  `db:"account_id"`
	Username  string `db:"username"`
}

func (u *User) CollectionName() string {
	return `users`
}

type AccountStore struct {
	bond.Store
}

func (s AccountStore) FindOne(cond db.Cond) (*Account, error) {
	var a *Account
	err := s.Find(cond).One(&a)
	return a, err
}

type UserStore struct {
	bond.Store
}

func init() {
	// os.Setenv("UPPERIO_DB_DEBUG", "1")

	var err error
	DB = &database{}

	DB.Session, err = bond.Open(`postgresql`, db.Settings{
		Host: "127.0.0.1", Database: "bond_test",
	})

	if err != nil {
		panic(err)
	}

	DB.Account = AccountStore{Store: DB.Store("accounts")}
	DB.User = UserStore{Store: DB.Store("users")}
}

func dbConnected() bool {
	if DB == nil {
		return false
	}
	err := DB.Ping()
	if err != nil {
		return false
	}
	return true
}

func dbReset() {
	cols, _ := DB.Collections()
	for _, k := range cols {
		col, err := DB.Collection(k)
		if err == nil {
			col.Truncate()
		}
	}
}

func TestMain(t *testing.M) {
	status := 0
	if dbConnected() {
		dbReset()
		status = t.Run()
	} else {
		status = -1
	}
	os.Exit(status)
}

func TestAccount(t *testing.T) {
	// -------
	// Create
	// -------
	user := &User{Username: "peter"}
	err := DB.Save(user)
	assert.NoError(t, err)

	acct := &Account{Name: "Pressly"}
	err = DB.Account.Save(acct)
	assert.NoError(t, err)

	// -------
	// Read
	// -------
	var acctChk *Account
	acctChk = &Account{}

	err = DB.Account.Find(db.Cond{"id": acct.ID}).One(&acctChk)
	assert.NoError(t, err)
	assert.Equal(t, acct.Name, acctChk.Name)

	err = DB.Find(db.Cond{"id": acct.ID}).One(acctChk)
	assert.NoError(t, err)
	assert.Equal(t, acct.Name, acctChk.Name)

	err = DB.Store("accounts").Find(db.Cond{"id": acct.ID}).One(acctChk)
	assert.NoError(t, err)
	assert.Equal(t, acct.Name, acctChk.Name)

	err = DB.Store(acctChk).Find(db.Cond{"id": acct.ID}).One(acctChk)
	assert.NoError(t, err)
	assert.Equal(t, acct.Name, acctChk.Name)

	colName := DB.Store("accounts").Name()
	assert.Equal(t, "accounts", colName)

	count, err := DB.Account.Find(db.Cond{}).Count()
	assert.NoError(t, err)
	assert.True(t, count == 1)

	count, err = DB.Account.Find().Count()
	assert.NoError(t, err)
	assert.True(t, count == 1)

	a, err := DB.Account.FindOne(db.Cond{"id": 1})
	assert.NoError(t, err)
	assert.NotNil(t, a)

	// -------
	// Update
	// -------
	acct.Disabled = true
	err = DB.Save(acct)
	assert.NoError(t, err)

	count, err = DB.Account.Find(db.Cond{}).Count()
	assert.NoError(t, err)
	assert.True(t, count == 1)

	// -------
	// Delete
	// -------
	err = DB.Delete(acct)
	assert.NoError(t, err)

	count, err = DB.Account.Find(db.Cond{}).Count()
	assert.NoError(t, err)
	assert.True(t, count == 0)
}

func TestDelete(t *testing.T) {
	acct := &Account{Name: "Pressly"}
	err := DB.Save(acct)
	assert.NoError(t, err)
	assert.True(t, acct.ID != 0)

	// Delete by query -- without callbacks
	err = DB.Account.Find(db.Cond{"name": acct.Name}).Remove()
	assert.NoError(t, err)

	err = DB.Account.Delete(&Account{Name: "X"})
	assert.Error(t, bond.ErrZeroItemID)
}

func TestSlices(t *testing.T) {
	id, err := DB.Account.Append(&Account{Name: "Apple"})
	assert.NoError(t, err)
	assert.True(t, id.(int64) > 0)

	id, err = DB.Account.Append(Account{Name: "Google"})
	assert.NoError(t, err)
	assert.True(t, id.(int64) > 0)

	var accts []*Account
	err = DB.Account.Find(db.Cond{}).All(&accts)
	assert.NoError(t, err)
	assert.Len(t, accts, 2)
}

func TestSelectOnlyIDs(t *testing.T) {
	var ids []struct {
		id int64 `db:"id"`
	}
	err := DB.Account.Find(db.Cond{}).Select("id").All(&ids)
	assert.NoError(t, err)
	assert.Len(t, ids, 2)
	assert.NotEmpty(t, ids[0])
}

// TODO:
// make a test with a join example...
