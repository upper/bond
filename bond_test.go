package bond_test

import (
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"database/sql"
	"github.com/stretchr/testify/assert"
	"upper.io/bond"
	"upper.io/db.v2"
	"upper.io/db.v2/postgresql"
)

var (
	DB           *database
	connSettings postgresql.ConnectionURL
)

func pickDefault(env string, def string) string {
	if v := os.Getenv(env); v != "" {
		return v
	}
	return def
}

type database struct {
	bond.Session

	Account AccountStore `collection:"accounts"`
	User    UserStore    `collection:"users"`
	Log     LogStore     `collection:"logs"`
}

type Log struct {
	ID      int64  `db:"id,omitempty,pk"`
	Message string `db:"message"`
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

func (a Account) AfterCreate(sess bond.Session) error {
	message := fmt.Sprintf("Account %q was created.", a.Name)
	return sess.Save(&Log{Message: message})
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

func (u User) AfterCreate(sess bond.Session) error {
	message := fmt.Sprintf("User %q was created.", u.Username)
	return sess.Save(&Log{Message: message})
}

func (u *User) CollectionName() string {
	return `users`
}

func (l *Log) CollectionName() string {
	return `logs`
}

type LogStore struct {
	bond.Store
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
	connSettings = postgresql.ConnectionURL{
		Host:     fmt.Sprintf("%s:%s", pickDefault("DB_HOST", "127.0.0.1"), pickDefault("DB_PORT", "5432")),
		User:     pickDefault("BOND_USER", "bond_user"),
		Database: pickDefault("BOND_DB", "bond_test"),
		Password: pickDefault("BOND_PASSWORD", "bond_password"),
	}

	var err error
	DB = &database{}

	DB.Session, err = bond.Open(postgresql.Adapter, connSettings)

	if err != nil {
		panic(err)
	}

	DB.Account = AccountStore{Store: DB.Store("accounts")}
	DB.User = UserStore{Store: DB.Store("users")}
	DB.Log = LogStore{Store: DB.Store("logs")}
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
		DB.Collection(k).Truncate()
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

	// Should fail because user is a UNIQUE value.
	err = DB.Save(&User{Username: "peter"})
	assert.Error(t, err)

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
	err = DB.Account.Find(db.Cond{"name": acct.Name}).Delete()
	assert.NoError(t, err)

	err = DB.Account.Delete(&Account{Name: "X"})
	assert.Error(t, bond.ErrZeroItemID)
}

func TestSlices(t *testing.T) {
	id, err := DB.Account.Insert(&Account{Name: "Apple"})
	assert.NoError(t, err)
	assert.True(t, id.(int64) > 0)

	id, err = DB.Account.Insert(Account{Name: "Google"})
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

func TestTransaction(t *testing.T) {
	var err error

	// This transaction should fail because user is a UNIQUE value and we already
	// have a "peter".
	err = DB.BondTx(func(sess bond.Session) error {
		return sess.Save(&User{Username: "peter"})
	})
	assert.Error(t, err)

	// This transaction should fail because user is a UNIQUE value and we already
	// have a "peter".
	err = DB.BondTx(func(sess bond.Session) error {
		return DB.User.With(sess).Save(&User{Username: "peter"})
	})
	assert.Error(t, err)

	// This transaction will have no errors, but we'll produce one in order for
	// it to rollback at the last moment.
	err = DB.BondTx(func(sess bond.Session) error {
		if err := DB.User.With(sess).Save(&User{Username: "Joe"}); err != nil {
			return err
		}

		if err := sess.Save(&User{Username: "Cool"}); err != nil {
			return err
		}

		return fmt.Errorf("Rolling back for no reason.")
	})
	assert.Error(t, err)

	// Attempt to add two new unique values, if the transaction above had not
	// been rolled back this transaction will fail.
	err = DB.BondTx(func(sess bond.Session) error {
		if err := DB.User.With(sess).Save(&User{Username: "Joe"}); err != nil {
			return err
		}

		if err := sess.Save(&User{Username: "Cool"}); err != nil {
			return err
		}

		return nil
	})
	assert.NoError(t, err)

	// If the transaction above was successful, this one will fail.
	err = DB.BondTx(func(sess bond.Session) error {
		if err := DB.User.With(sess).Save(&User{Username: "Joe"}); err != nil {
			return err
		}

		if err := sess.Save(&User{Username: "Cool"}); err != nil {
			return err
		}

		return nil
	})
	assert.Error(t, err)
}

func TestInheritedTransaction(t *testing.T) {
	anoterSess, err := bond.Open(postgresql.Adapter, connSettings)
	assert.NoError(t, err)
	defer anoterSess.Close()

	// This is a normal SQL database.
	drv := anoterSess.Driver()
	sqlDB := drv.(*sql.DB)

	// We create a transaction on this SQL database.
	sqlTx, err := sqlDB.Begin()
	assert.NoError(t, err)

	// And pass that transaction to bond, this whole session is a transaction.
	sess, err := bond.New(postgresql.Adapter, sqlTx)
	assert.NoError(t, err)

	// Should fail because user is a UNIQUE value and we already have a "peter".
	err = DB.User.With(sess).Save(&User{Username: "peter"})
	assert.Error(t, err)

	// The transaction is controlled outside bond.
	err = sqlTx.Rollback()
	assert.NoError(t, err)

	// The sqlTx is worthless now.
	err = DB.User.With(sess).Save(&User{Username: "peter-2"})
	assert.Error(t, err)

	// But we can create a new one.
	sqlTx, err = sqlDB.Begin()
	assert.NoError(t, err)
	assert.NotNil(t, sqlTx)

	// And create another bond session.
	sess, err = bond.New(postgresql.Adapter, sqlTx)
	assert.NoError(t, err)

	// This model uses the given session to do stuff.
	userTx := DB.User.With(sess)

	// Adding two new values.
	err = userTx.Save(&User{Username: "Joe-2"})
	assert.NoError(t, err)

	err = userTx.Save(&User{Username: "Cool-2"})
	assert.NoError(t, err)

	// And a value that is going to be rolled back.
	err = DB.Account.With(sess).Save(&Account{Name: "Rolled back"})
	assert.NoError(t, err)

	// This session happens to be a transaction, let's rollback everything.
	err = sqlTx.Rollback()
	assert.NoError(t, err)

	// Start again.
	sqlTx, err = sqlDB.Begin()
	assert.NoError(t, err)

	sess, err = bond.New(postgresql.Adapter, sqlTx)
	assert.NoError(t, err)

	userTx = DB.User.With(sess)

	// Attempt to add two unique values.
	err = userTx.Save(&User{Username: "Joe-2"})
	assert.NoError(t, err)

	err = userTx.Save(&User{Username: "Cool-2"})
	assert.NoError(t, err)

	// And a value that is going to be commited.
	err = DB.Account.With(sess).Save(&Account{Name: "Commited!"})
	assert.NoError(t, err)

	// Yes, commit them.
	err = sqlTx.Commit()
	assert.NoError(t, err)
}
