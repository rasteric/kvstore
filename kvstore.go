package kvstore

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"sync/atomic"

	"github.com/jmoiron/sqlx"
	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)

var NotFoundErr = errors.New(`key not found`)
var NotOpenErr = errors.New(`key value store is closed`)
var AlreadyOpenErr = errors.New(`database already open`)
var NoDefaultErr = errors.New(`no default value set for given key`)

// KeyValueStore is the interface for a key value database.
type KeyValueStore interface {
	Open(path string) error                   // open the database at directory path
	Close() error                             // close the database
	Set(key string, value any) error          // set the key to the given value, which must be gob serializable
	Get(key string) (any, error)              // get the value for key, NotFoundErr if there is no key
	SetMany(map[string]any) error             // set all key-value pairs in the map in one transaction
	GetAll(limit int) (map[string]any, error) // get all key-value pairs as a map
	Revert(key string) error                  // revert key to its default
	Info(key string) (KeyInfo, bool)          // returns key information for a key if it is present
	Delete(key string) error                  // remove the key and value for the key
	DeleteMany(keys []string) error           // remove all the given keys in one transaction
	SetDefault(key string,                    // set a default and info for a key
		value any,
		info KeyInfo) error
}

// KeyInfo is provides information about a key. This is useful for preference systems.
type KeyInfo struct {
	Description string
	Category    string
}

// KVStore implements KvStore interface with an sqlite database backend.
type KVStore struct {
	path  string
	sqx   *sqlx.DB
	sq    *sql.DB
	state uint32
}

// New creates a new key value store that is not yet opened.
func New() *KVStore {
	return &KVStore{}
}

var _ KeyValueStore = (*KVStore)(nil)

// Open a database at the path specified when the database was created,
// which holds all database files. If directories to path/name do not exist, they are created
// recursively with Unix permissions 0755.
func (db *KVStore) Open(path string) error {
	if atomic.LoadUint32(&db.state) > 255 {
		return AlreadyOpenErr
	}
	db.path = path
	var err error
	if db.path == "" {
		db.path, err = os.Getwd()
		if err != nil {
			return err
		}
	}
	_, err = os.Stat(db.path)
	if err != nil {
		if os.IsNotExist(err) {
			err := os.MkdirAll(db.path, 0755)
			if err != nil {
				return err
			}
		}
	}
	file := filepath.Join(db.path, "kvstore.sqlite")
	db.path = file
	db.sq, err = sql.Open("sqlite3", file)
	if err != nil {
		return err
	}
	db.sqx = sqlx.NewDb(db.sq, "sqlite3")
	if err != nil {
		return err
	}
	return db.init()
}

// init initializes the database tables if necessary.
func (db *KVStore) init() error {
	_, err := db.sqx.Exec(`
PRAGMA journal_mode=WAL;
PRAGMA synchronous=NORMAL;
PRAGMA auto_vacuum=FULL;
PRAGMA journal_size_limit = 67108864;
PRAGMA mmap_size = 134217728;
PRAGMA cache_size = 2000;
PRAGMA busy_timeout = 5000;

CREATE TABLE IF NOT EXISTS kv(
  key TEXT PRIMARY KEY NOT NULL,
  value BLOB,
  original BLOB,
  info TEXT,
  category TEXT
);
`)
	if err != nil {
		atomic.StoreUint32(&db.state, 3)
		return err
	}
	atomic.StoreUint32(&db.state, 256)
	return nil
}

// Close closes the database.
func (db *KVStore) Close() error {
	if atomic.LoadUint32(&db.state) < 256 {
		return nil
	}
	atomic.StoreUint32(&db.state, 2)
	err := db.sqx.Close()
	if err != nil {
		atomic.StoreUint32(&db.state, 3)
	}
	return err
}

// SetDefault sets a default value for the given key, as well as info and category.
func (db *KVStore) SetDefault(key string, value any, info KeyInfo) error {
	if atomic.LoadUint32(&db.state) < 256 {
		return NotOpenErr
	}
	original, err := MarshalBinary(value)
	if err != nil {
		return err
	}
	_, err = db.sqx.Exec(`INSERT INTO kv(key,original,info,category) VALUES(?,?,?,?) ON CONFLICT(key) DO UPDATE SET original=?,info=?,category=?;`,
		key, original, info.Description, info.Category, original, info.Description, info.Category)
	return err
}

// Set sets the value for the given key, overwriting an existing value for the key if there is one.
func (db *KVStore) Set(key string, value any) error {
	if atomic.LoadUint32(&db.state) < 256 {
		return NotOpenErr
	}
	b, err := MarshalBinary(value)
	if err != nil {
		return err
	}
	_, err = db.sqx.Exec(`INSERT INTO kv(key,value) VALUES(?,?) ON CONFLICT(key) DO UPDATE SET value=?;`,
		key, b, b)
	return err
}

// SetMany sets all pairs in the given map in one transaction.
func (db *KVStore) SetMany(pairs map[string]any) error {
	if atomic.LoadUint32(&db.state) < 256 {
		return NotOpenErr
	}
	tx, err := db.sqx.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for k, v := range pairs {
		b, err := MarshalBinary(v)
		if err != nil {
			return err
		}
		_, err = tx.Exec(`INSERT INTO kv(key,value) VALUES(?,?) ON CONFLICT(key) DO UPDATE SET value=?;`, k, b, b)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

// Get gets the value for the given key, the default if no value for the key is stored but a default is
// present, and NotFoundErr if neither of them is present.
func (db *KVStore) Get(key string) (any, error) {
	if atomic.LoadUint32(&db.state) < 256 {
		return nil, NotOpenErr
	}
	var b []byte
	err := db.sqx.Get(&b, `SELECT value FROM kv WHERE key=? LIMIT 1;`, key)
	if err != nil || b == nil {
		return db.getDefault(key)
	}
	return UnmarshalBinary(b)
}

// GetAll returns all key-value pairs as a map. If limit is 0 or negative, all key value pairs are returned.
// Although this is usually not advisable, this method may be used in combination with SetMany to save and
// load maps, i.e., use the key value store merely for persistence and keep the data in memory.
func (db *KVStore) GetAll(limit int) (map[string]any, error) {
	if atomic.LoadUint32(&db.state) < 256 {
		return nil, NotOpenErr
	}
	var rows *sqlx.Rows
	var err error
	if limit <= 0 {
		rows, err = db.sqx.Queryx(`SELECT key,value,original FROM kv ORDER BY key ASC;`)
	} else {
		rows, err = db.sqx.Queryx(`SELECT key,value,original FROM kv ORDER BY key ASC LIMIT ?;`, limit)
	}
	if err != nil {
		return nil, err
	}
	if rows == nil {
		return nil, NotFoundErr
	}
	result := make(map[string]any)
	for rows.Next() {
		var key string
		var value, original []byte
		err = rows.Scan(&key, &value, &original)
		if err != nil {
			return result, err
		}
		if value != nil {
			v, err2 := UnmarshalBinary(value)
			if err != nil {
				err = errors.Join(err, err2)
			} else {
				result[key] = v
			}
		} else if original != nil {
			v, err2 := UnmarshalBinary(original)
			if err != nil {
				err = errors.Join(err, err2)
			} else {
				result[key] = v
			}
		}
	}
	return result, err
}

// getDefault obtains the default for the given key, ErrNotFound if there is none.
func (db *KVStore) getDefault(key string) (any, error) {
	if atomic.LoadUint32(&db.state) < 256 {
		return nil, NotOpenErr
	}
	var b []byte
	err := db.sqx.Get(&b, `SELECT original FROM kv WHERE key=? LIMIT 1;`, key)
	if errors.Is(err, sql.ErrNoRows) || b == nil {
		return nil, NotFoundErr
	}
	return UnmarshalBinary(b)
}

// Info attempts to obtain information about the given key, returns false if none can be found.
// This method will also return false if an error occurs.
func (db *KVStore) Info(key string) (KeyInfo, bool) {
	var info KeyInfo
	if atomic.LoadUint32(&db.state) < 256 {
		return info, false
	}
	row := db.sqx.QueryRowx(`SELECT info,category FROM kv WHERE key=? LIMIT 1;`, key)
	if row == nil {
		return info, false
	}
	err := row.Scan(&info.Description, &info.Category)
	if err != nil {
		return info, false
	}
	return info, true
}

// Revert reverts the value for the given key to its default. If no default has been set, NoDefaultErr is returned.
func (db *KVStore) Revert(key string) error {
	if atomic.LoadUint32(&db.state) < 256 {
		return NotOpenErr
	}
	_, err := db.sqx.Exec(`UPDATE kv SET value=original WHERE key=?;`, key)
	if err != nil {
		return NoDefaultErr
	}
	return nil
}

// Delete removes the key and value from the key value store.
func (db *KVStore) Delete(key string) error {
	if atomic.LoadUint32(&db.state) < 256 {
		return NotOpenErr
	}
	_, err := db.sqx.Exec(`DELETE FROM kv WHERE key=?;`, key)
	return err
}

// DeleteMany removes all given keys in one transaction.
func (db *KVStore) DeleteMany(keys []string) error {
	if atomic.LoadUint32(&db.state) < 256 {
		return NotOpenErr
	}
	tx, err := db.sqx.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, k := range keys {
		_, err = tx.Exec(`DELETE FROM kv WHERE key=?;`, k)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}
