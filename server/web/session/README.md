session
==============

session is a Go session manager. It can use many session providers. Just like the `database/sql`
and `database/sql/driver`.

## How to install?

	go get github.com/asish-tom/beego/v2/server/web/session

## What providers are supported?

As of now this session manager support memory, file, Redis and MySQL.

## How to use it?

First you must import it

	import (
		"github.com/asish-tom/beego/v2/server/web/session"
	)

Then in you web app init the global session manager

	var globalSessions *session.Manager

* Use **memory** as provider:

  	func init() {
  		globalSessions, _ = session.NewManager("memory", `{"cookieName":"gosessionid","gclifetime":3600}`)
  		go globalSessions.GC()
  	}

* Use **file** as provider, the last param is the path where you want file to be stored:

  	func init() {
  		globalSessions, _ = session.NewManager("file",`{"cookieName":"gosessionid","gclifetime":3600,"ProviderConfig":"./tmp"}`)
  		go globalSessions.GC()
  	}

* Use **Redis** as provider, the last param is the Redis conn address,poolsize,password:

  	func init() {
  		globalSessions, _ = session.NewManager("redis", `{"cookieName":"gosessionid","gclifetime":3600,"ProviderConfig":"127.0.0.1:6379,100,astaxie"}`)
  		go globalSessions.GC()
  	}

* Use **MySQL** as provider, the last param is the DSN, learn more
  from [mysql](https://github.com/go-sql-driver/mysql#dsn-data-source-name):

  	func init() {
  		globalSessions, _ = session.NewManager(
  			"mysql", `{"cookieName":"gosessionid","gclifetime":3600,"ProviderConfig":"username:password@protocol(address)/dbname?param=value"}`)
  		go globalSessions.GC()
  	}

* Use **Cookie** as provider:

  	func init() {
  		globalSessions, _ = session.NewManager(
  			"cookie", `{"cookieName":"gosessionid","enableSetCookie":false,"gclifetime":3600,"ProviderConfig":"{\"cookieName\":\"gosessionid\",\"securityKey\":\"beegocookiehashkey\"}"}`)
  		go globalSessions.GC()
  	}

Finally in the handlerfunc you can use it like this

	func login(w http.ResponseWriter, r *http.Request) {
		sess := globalSessions.SessionStart(w, r)
		defer sess.SessionRelease(w)
		username := sess.Get("username")
		fmt.Println(username)
		if r.Method == "GET" {
			t, _ := template.ParseFiles("login.gtpl")
			t.Execute(w, nil)
		} else {
			fmt.Println("username:", r.Form["username"])
			sess.Set("username", r.Form["username"])
			fmt.Println("password:", r.Form["password"])
		}
	}

## How to write own provider?

When you develop a web app, maybe you want to write own provider because you must meet the requirements.

Writing a provider is easy. You only need to define two struct types
(Store and Provider), which satisfy the interface definition. Maybe you will find the **memory** provider is a good
example.

	// Store contains all data for one session process with specific id.
    type Store interface {
        Set(ctx context.Context, key, value interface{}) error              // Set set session value
        Get(ctx context.Context, key interface{}) interface{}               // Get get session value
        Delete(ctx context.Context, key interface{}) error                  // Delete delete session value
        SessionID(ctx context.Context) string                               // SessionID return current sessionID
        SessionReleaseIfPresent(ctx context.Context, w http.ResponseWriter) // SessionReleaseIfPresent release the resource & save data to provider & return the data when the session is present, not all implementation support this feature, you need to check if the specific implementation if support this feature.
        SessionRelease(ctx context.Context, w http.ResponseWriter)          // SessionRelease release the resource & save data to provider & return the data
        Flush(ctx context.Context) error                                    // Flush delete all data
    }
	
	type Provider interface {
		SessionInit(gclifetime int64, config string) error
		SessionRead(sid string) (SessionStore, error)
		SessionExist(sid string) (bool, error)
		SessionRegenerate(oldsid, sid string) (SessionStore, error)
		SessionDestroy(sid string) error
		SessionAll() int //get all active session
		SessionGC()
	}

## LICENSE

BSD License http://creativecommons.org/licenses/BSD/
