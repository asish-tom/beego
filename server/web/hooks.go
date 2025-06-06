package web

import (
	"encoding/json"
	"mime"
	"net/http"
	"path/filepath"

	"github.com/asish-tom/beego/v2/core/logs"
	"github.com/asish-tom/beego/v2/server/web/context"
	"github.com/asish-tom/beego/v2/server/web/session"
)

// register MIME type with content type
func registerMime() error {
	for k, v := range mimemaps {
		mime.AddExtensionType(k, v)
	}
	return nil
}

// register default error http handlers, 404,401,403,500 and 503.
func registerDefaultErrorHandler() error {
	m := map[string]func(http.ResponseWriter, *http.Request){
		"401": unauthorized,
		"402": paymentRequired,
		"403": forbidden,
		"404": notFound,
		"405": methodNotAllowed,
		"500": internalServerError,
		"501": notImplemented,
		"502": badGateway,
		"503": serviceUnavailable,
		"504": gatewayTimeout,
		"417": invalidxsrf,
		"422": missingxsrf,
		"413": payloadTooLarge,
	}
	for e, h := range m {
		if _, ok := ErrorMaps[e]; !ok {
			ErrorHandler(e, h)
		}
	}
	return nil
}

func registerSession() error {
	if BConfig.WebConfig.Session.SessionOn {
		var err error
		sessionConfig, err := AppConfig.String("sessionConfig")
		conf := new(session.ManagerConfig)
		if sessionConfig == "" || err != nil {
			conf.CookieName = BConfig.WebConfig.Session.SessionName
			conf.EnableSetCookie = BConfig.WebConfig.Session.SessionAutoSetCookie
			conf.Gclifetime = BConfig.WebConfig.Session.SessionGCMaxLifetime
			conf.Secure = BConfig.Listen.EnableHTTPS
			conf.CookieLifeTime = BConfig.WebConfig.Session.SessionCookieLifeTime
			conf.ProviderConfig = filepath.ToSlash(BConfig.WebConfig.Session.SessionProviderConfig)
			conf.DisableHTTPOnly = BConfig.WebConfig.Session.SessionDisableHTTPOnly
			conf.Domain = BConfig.WebConfig.Session.SessionDomain
			conf.EnableSidInHTTPHeader = BConfig.WebConfig.Session.SessionEnableSidInHTTPHeader
			conf.SessionNameInHTTPHeader = BConfig.WebConfig.Session.SessionNameInHTTPHeader
			conf.EnableSidInURLQuery = BConfig.WebConfig.Session.SessionEnableSidInURLQuery
			conf.CookieSameSite = BConfig.WebConfig.Session.SessionCookieSameSite
			conf.SessionIDPrefix = BConfig.WebConfig.Session.SessionIDPrefix
		} else {
			if err = json.Unmarshal([]byte(sessionConfig), conf); err != nil {
				return err
			}
		}
		if GlobalSessions, err = session.NewManager(BConfig.WebConfig.Session.SessionProvider, conf); err != nil {
			return err
		}
		go GlobalSessions.GC()
	}
	return nil
}

func registerTemplate() error {
	defer lockViewPaths()
	if err := AddViewPath(BConfig.WebConfig.ViewsPath); err != nil {
		if BConfig.RunMode == DEV {
			logs.Warn(err)
		}
		return err
	}
	return nil
}

func registerGzip() error {
	if BConfig.EnableGzip {
		context.InitGzip(
			AppConfig.DefaultInt("gzipMinLength", -1),
			AppConfig.DefaultInt("gzipCompressLevel", -1),
			AppConfig.DefaultStrings("includedMethods", []string{"GET"}),
		)
	}
	return nil
}
