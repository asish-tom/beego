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

// Package apiauth provides handlers to enable apiauth support.
//
// Simple Usage:
//
//	import(
//		"github.com/asish-tom/beego/v2"
//		"github.com/asish-tom/beego/v2/server/web/filter/apiauth"
//	)
//
//	func main(){
//		// apiauth every request
//		beego.InsertFilter("*", beego.BeforeRouter,apiauth.APIBasicAuth("appid","appkey"))
//		beego.Run()
//	}
//
// Advanced Usage:
//
//	func getAppSecret(appid string) string {
//		// get appsecret by appid
//		// maybe store in configure, maybe in database
//	}
//
//	beego.InsertFilter("*", beego.BeforeRouter,apiauth.APISecretAuth(getAppSecret, 360))
//
// Information:
//
// # In the request user should include these params in the query
//
// 1. appid
//
//	appid is assigned to the application
//
// 2. signature
//
//	get the signature use apiauth.Signature()
//
//	when you send to server remember use url.QueryEscape()
//
// 3. timestamp:
//
//	send the request time, the format is yyyy-mm-dd HH:ii:ss
package apiauth

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/url"
	"sort"
	"time"

	"github.com/asish-tom/beego/v2/server/web"
	"github.com/asish-tom/beego/v2/server/web/context"
)

// AppIDToAppSecret gets appsecret through appid
type AppIDToAppSecret func(string) string

// APIBasicAuth uses the basic appid/appkey as the AppIdToAppSecret
func APIBasicAuth(appid, appkey string) web.FilterFunc {
	ft := func(aid string) string {
		if aid == appid {
			return appkey
		}
		return ""
	}
	return APISecretAuth(ft, 300)
}

// APISecretAuth uses AppIdToAppSecret verify and
func APISecretAuth(f AppIDToAppSecret, timeout int) web.FilterFunc {
	return func(ctx *context.Context) {
		if ctx.Input.Query("appid") == "" {
			ctx.ResponseWriter.WriteHeader(403)
			ctx.WriteString("missing query parameter: appid")
			return
		}
		appsecret := f(ctx.Input.Query("appid"))
		if appsecret == "" {
			ctx.ResponseWriter.WriteHeader(403)
			ctx.WriteString("appid query parameter missing")
			return
		}
		if ctx.Input.Query("signature") == "" {
			ctx.ResponseWriter.WriteHeader(403)
			ctx.WriteString("missing query parameter: signature")

			return
		}
		if ctx.Input.Query("timestamp") == "" {
			ctx.ResponseWriter.WriteHeader(403)
			ctx.WriteString("missing query parameter: timestamp")
			return
		}
		u, err := time.Parse("2006-01-02 15:04:05", ctx.Input.Query("timestamp"))
		if err != nil {
			ctx.ResponseWriter.WriteHeader(403)
			ctx.WriteString("incorrect timestamp format. Should be in the form 2006-01-02 15:04:05")

			return
		}
		t := time.Now()
		if t.Sub(u).Seconds() > float64(timeout) {
			ctx.ResponseWriter.WriteHeader(403)
			ctx.WriteString("request timer timeout exceeded. Please try again")
			return
		}
		if ctx.Input.Query("signature") !=
			Signature(appsecret, ctx.Input.Method(), ctx.Request.Form, ctx.Input.URL()) {
			ctx.ResponseWriter.WriteHeader(403)
			ctx.WriteString("authentication failed")
		}
	}
}

// Signature generates signature with appsecret/method/params/RequestURI
func Signature(appsecret, method string, params url.Values, RequestURL string) (result string) {
	var b bytes.Buffer
	keys := make([]string, 0, len(params))
	pa := make(map[string]string)
	for k, v := range params {
		pa[k] = v[0]
		keys = append(keys, k)
	}

	sort.Strings(keys)

	for _, key := range keys {
		if key == "signature" {
			continue
		}

		val := pa[key]
		if key != "" && val != "" {
			b.WriteString(key)
			b.WriteString(val)
		}
	}

	stringToSign := fmt.Sprintf("%v\n%v\n%v\n", method, b.String(), RequestURL)

	sha256 := sha256.New
	hash := hmac.New(sha256, []byte(appsecret))
	hash.Write([]byte(stringToSign))
	return base64.StdEncoding.EncodeToString(hash.Sum(nil))
}
