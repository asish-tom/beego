// Copyright 2014 beego Author. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package postgres for session provider
//
// depends on github.com/lib/pq:
//
// go install github.com/lib/pq
//
// needs this table in your database:
//
// CREATE TABLE session (
// session_key	char(64) NOT NULL,
// session_data	bytea,
// session_expiry	timestamp NOT NULL,
// CONSTRAINT session_key PRIMARY KEY(session_key)
// );
//
// will be activated with these settings in app.conf:
//
// SessionOn = true
// SessionProvider = postgresql
// SessionSavePath = "user=a password=b dbname=c sslmode=disable"
// SessionName = session
//
// Usage:
// import(
//
//	_ "github.com/asish-tom/beego/v2/server/web/session/postgresql"
//	"github.com/asish-tom/beego/v2/server/web/session"
//
// )
//
//	func init() {
//		globalSessions, _ = session.NewManager("postgresql", ``{"cookieName":"gosessionid","gclifetime":3600,"ProviderConfig":"user=pqgotest dbname=pqgotest sslmode=verify-full"}``)
//		go globalSessions.GC()
//	}
package postgres

import (
	"context"
	"database/sql"
	"net/http"
	"sync"
	"time"

	_ "github.com/lib/pq"

	"github.com/asish-tom/beego/v2/server/web/session"
)

var postgresqlpder = &Provider{}

// SessionStore postgresql session store
type SessionStore struct {
	c      *sql.DB
	sid    string
	lock   sync.RWMutex
	values map[interface{}]interface{}
}

// Set value in postgresql session.
// it is temp value in map.
func (st *SessionStore) Set(ctx context.Context, key, value interface{}) error {
	st.lock.Lock()
	defer st.lock.Unlock()
	st.values[key] = value
	return nil
}

// Get value from postgresql session
func (st *SessionStore) Get(ctx context.Context, key interface{}) interface{} {
	st.lock.RLock()
	defer st.lock.RUnlock()
	if v, ok := st.values[key]; ok {
		return v
	}
	return nil
}

// Delete value in postgresql session
func (st *SessionStore) Delete(ctx context.Context, key interface{}) error {
	st.lock.Lock()
	defer st.lock.Unlock()
	delete(st.values, key)
	return nil
}

// Flush clear all values in postgresql session
func (st *SessionStore) Flush(context.Context) error {
	st.lock.Lock()
	defer st.lock.Unlock()
	st.values = make(map[interface{}]interface{})
	return nil
}

// SessionID get session id of this postgresql session store
func (st *SessionStore) SessionID(context.Context) string {
	return st.sid
}

// SessionRelease save postgresql session values to database.
// must call this method to save values to database.
func (st *SessionStore) SessionRelease(_ context.Context, _ http.ResponseWriter) {
	defer st.c.Close()
	st.lock.RLock()
	values := st.values
	st.lock.RUnlock()
	b, err := session.EncodeGob(values)
	if err != nil {
		return
	}
	st.c.Exec("UPDATE session set session_data=$1, session_expiry=$2 where session_key=$3",
		b, time.Now().Format(time.RFC3339), st.sid)
}

// SessionReleaseIfPresent save postgresql session values to database when key is present
func (st *SessionStore) SessionReleaseIfPresent(ctx context.Context, w http.ResponseWriter) {
	st.SessionRelease(ctx, w)
}

// Provider postgresql session provider
type Provider struct {
	maxlifetime int64
	savePath    string
}

// connect to postgresql
func (mp *Provider) connectInit() *sql.DB {
	db, e := sql.Open("postgres", mp.savePath)
	if e != nil {
		return nil
	}
	return db
}

// SessionInit init postgresql session.
// savepath is the connection string of postgresql.
func (mp *Provider) SessionInit(ctx context.Context, maxlifetime int64, savePath string) error {
	mp.maxlifetime = maxlifetime
	mp.savePath = savePath
	return nil
}

// SessionRead get postgresql session by sid
func (mp *Provider) SessionRead(ctx context.Context, sid string) (session.Store, error) {
	c := mp.connectInit()
	row := c.QueryRow("select session_data from session where session_key=$1", sid)
	var sessiondata []byte
	err := row.Scan(&sessiondata)
	if err == sql.ErrNoRows {
		_, err = c.Exec("insert into session(session_key,session_data,session_expiry) values($1,$2,$3)",
			sid, "", time.Now().Format(time.RFC3339))

		if err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}

	var kv map[interface{}]interface{}
	if len(sessiondata) == 0 {
		kv = make(map[interface{}]interface{})
	} else {
		kv, err = session.DecodeGob(sessiondata)
		if err != nil {
			return nil, err
		}
	}
	rs := &SessionStore{c: c, sid: sid, values: kv}
	return rs, nil
}

// SessionExist check postgresql session exist
func (mp *Provider) SessionExist(ctx context.Context, sid string) (bool, error) {
	c := mp.connectInit()
	defer c.Close()
	row := c.QueryRow("select session_data from session where session_key=$1", sid)
	var sessiondata []byte
	err := row.Scan(&sessiondata)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// SessionRegenerate generate new sid for postgresql session
func (mp *Provider) SessionRegenerate(ctx context.Context, oldsid, sid string) (session.Store, error) {
	c := mp.connectInit()
	row := c.QueryRow("select session_data from session where session_key=$1", oldsid)
	var sessiondata []byte
	err := row.Scan(&sessiondata)
	if err == sql.ErrNoRows {
		c.Exec("insert into session(session_key,session_data,session_expiry) values($1,$2,$3)",
			oldsid, "", time.Now().Format(time.RFC3339))
	}
	c.Exec("update session set session_key=$1 where session_key=$2", sid, oldsid)
	var kv map[interface{}]interface{}
	if len(sessiondata) == 0 {
		kv = make(map[interface{}]interface{})
	} else {
		kv, err = session.DecodeGob(sessiondata)
		if err != nil {
			return nil, err
		}
	}
	rs := &SessionStore{c: c, sid: sid, values: kv}
	return rs, nil
}

// SessionDestroy delete postgresql session by sid
func (mp *Provider) SessionDestroy(ctx context.Context, sid string) error {
	c := mp.connectInit()
	c.Exec("DELETE FROM session where session_key=$1", sid)
	c.Close()
	return nil
}

// SessionGC delete expired values in postgresql session
func (mp *Provider) SessionGC(context.Context) {
	c := mp.connectInit()
	c.Exec("DELETE from session where EXTRACT(EPOCH FROM (current_timestamp - session_expiry)) > $1", mp.maxlifetime)
	c.Close()
}

// SessionAll count values in postgresql session
func (mp *Provider) SessionAll(context.Context) int {
	c := mp.connectInit()
	defer c.Close()
	var total int
	err := c.QueryRow("SELECT count(*) as num from session").Scan(&total)
	if err != nil {
		return 0
	}
	return total
}

func init() {
	session.Register("postgresql", postgresqlpder)
}
