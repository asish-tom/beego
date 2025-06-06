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

package web

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/asish-tom/beego/v2/core/logs"
	"github.com/asish-tom/beego/v2/server/web/context"
)

type PrefixTestController struct {
	Controller
}

func (ptc *PrefixTestController) PrefixList() {
	ptc.Ctx.Output.Body([]byte("i am list in prefix test"))
}

type TestControllerWithInterface struct{}

func (m TestControllerWithInterface) Ping() {
	fmt.Println("pong")
}

func (m *TestControllerWithInterface) PingPointer() {
	fmt.Println("pong pointer")
}

type TestController struct {
	Controller
}

func (tc *TestController) Get() {
	tc.Data["Username"] = "astaxie"
	tc.Ctx.Output.Body([]byte("ok"))
}

func (tc *TestController) Post() {
	tc.Ctx.Output.Body([]byte(tc.Ctx.Input.Query(":name")))
}

func (tc *TestController) Param() {
	tc.Ctx.Output.Body([]byte(tc.Ctx.Input.Query(":name")))
}

func (tc *TestController) List() {
	tc.Ctx.Output.Body([]byte("i am list"))
}

func (tc *TestController) Params() {
	tc.Ctx.Output.Body([]byte(tc.Ctx.Input.Param("0") + tc.Ctx.Input.Param("1") + tc.Ctx.Input.Param("2")))
}

func (tc *TestController) Myext() {
	tc.Ctx.Output.Body([]byte(tc.Ctx.Input.Param(":ext")))
}

func (tc *TestController) GetURL() {
	tc.Ctx.Output.Body([]byte(tc.URLFor(".Myext")))
}

func (tc *TestController) GetParams() {
	tc.Ctx.WriteString(tc.Ctx.Input.Query(":last") + "+" +
		tc.Ctx.Input.Query(":first") + "+" + tc.Ctx.Input.Query("learn"))
}

func (tc *TestController) GetManyRouter() {
	tc.Ctx.WriteString(tc.Ctx.Input.Query(":id") + tc.Ctx.Input.Query(":page"))
}

func (tc *TestController) GetEmptyBody() {
	var res []byte
	tc.Ctx.Output.Body(res)
}

type JSONController struct {
	Controller
}

func (jc *JSONController) Prepare() {
	jc.Data["json"] = "prepare"
	jc.ServeJSON(true)
}

func (jc *JSONController) Get() {
	jc.Data["Username"] = "astaxie"
	jc.Ctx.Output.Body([]byte("ok"))
}

func TestPrefixUrlFor(t *testing.T) {
	handler := NewControllerRegister()
	handler.Add("/my/prefix/list", &PrefixTestController{}, WithRouterMethods(&PrefixTestController{}, "get:PrefixList"))

	if a := handler.URLFor(`PrefixTestController.PrefixList`); a != `/my/prefix/list` {
		logs.Info(a)
		t.Errorf("PrefixTestController.PrefixList must equal to /my/prefix/list")
	}
	if a := handler.URLFor(`TestController.PrefixList`); a != `` {
		logs.Info(a)
		t.Errorf("TestController.PrefixList must equal to empty string")
	}
}

func TestUrlFor(t *testing.T) {
	handler := NewControllerRegister()
	handler.Add("/api/list", &TestController{}, WithRouterMethods(&TestController{}, "*:List"))
	handler.Add("/person/:last/:first", &TestController{}, WithRouterMethods(&TestController{}, "*:Param"))
	if a := handler.URLFor("TestController.List"); a != "/api/list" {
		logs.Info(a)
		t.Errorf("TestController.List must equal to /api/list")
	}
	if a := handler.URLFor("TestController.Param", ":last", "xie", ":first", "asta"); a != "/person/xie/asta" {
		t.Errorf("TestController.Param must equal to /person/xie/asta, but get " + a)
	}
}

func TestUrlFor3(t *testing.T) {
	handler := NewControllerRegister()
	handler.AddAuto(&TestController{})
	a := handler.URLFor("TestController.Myext")
	if a != "/test/myext" && a != "/Test/Myext" {
		t.Errorf("TestController.Myext must equal to /test/myext, but get " + a)
	}
	a = handler.URLFor("TestController.GetURL")
	if a != "/test/geturl" && a != "/Test/GetURL" {
		t.Errorf("TestController.GetURL must equal to /test/geturl, but get " + a)
	}
}

func TestUrlFor2(t *testing.T) {
	handler := NewControllerRegister()
	handler.Add("/v1/:v/cms_:id(.+)_:page(.+).html", &TestController{}, WithRouterMethods(&TestController{}, "*:List"))
	handler.Add("/v1/:username/edit", &TestController{}, WithRouterMethods(&TestController{}, "get:GetURL"))
	handler.Add("/v1/:v(.+)_cms/ttt_:id(.+)_:page(.+).html", &TestController{}, WithRouterMethods(&TestController{}, "*:Param"))
	handler.Add("/:year:int/:month:int/:title/:entid", &TestController{})
	if handler.URLFor("TestController.GetURL", ":username", "astaxie") != "/v1/astaxie/edit" {
		logs.Info(handler.URLFor("TestController.GetURL"))
		t.Errorf("TestController.List must equal to /v1/astaxie/edit")
	}

	if handler.URLFor("TestController.List", ":v", "za", ":id", "12", ":page", "123") !=
		"/v1/za/cms_12_123.html" {
		logs.Info(handler.URLFor("TestController.List"))
		t.Errorf("TestController.List must equal to /v1/za/cms_12_123.html")
	}
	if handler.URLFor("TestController.Param", ":v", "za", ":id", "12", ":page", "123") !=
		"/v1/za_cms/ttt_12_123.html" {
		logs.Info(handler.URLFor("TestController.Param"))
		t.Errorf("TestController.List must equal to /v1/za_cms/ttt_12_123.html")
	}
	if handler.URLFor("TestController.Get", ":year", "1111", ":month", "11",
		":title", "aaaa", ":entid", "aaaa") !=
		"/1111/11/aaaa/aaaa" {
		logs.Info(handler.URLFor("TestController.Get"))
		t.Errorf("TestController.Get must equal to /1111/11/aaaa/aaaa")
	}
}

func TestUserFunc(t *testing.T) {
	r, _ := http.NewRequest("GET", "/api/list", nil)
	w := httptest.NewRecorder()

	handler := NewControllerRegister()
	handler.Add("/api/list", &TestController{}, WithRouterMethods(&TestController{}, "*:List"))
	handler.ServeHTTP(w, r)
	if w.Body.String() != "i am list" {
		t.Errorf("user define func can't run")
	}
}

func TestPostFunc(t *testing.T) {
	r, _ := http.NewRequest("POST", "/astaxie", nil)
	w := httptest.NewRecorder()

	handler := NewControllerRegister()
	handler.Add("/:name", &TestController{})
	handler.ServeHTTP(w, r)
	if w.Body.String() != "astaxie" {
		t.Errorf("post func should astaxie")
	}
}

func TestAutoFunc(t *testing.T) {
	r, _ := http.NewRequest("GET", "/test/list", nil)
	w := httptest.NewRecorder()

	handler := NewControllerRegister()
	handler.AddAuto(&TestController{})
	handler.ServeHTTP(w, r)
	if w.Body.String() != "i am list" {
		t.Errorf("user define func can't run")
	}
}

func TestAutoFunc2(t *testing.T) {
	r, _ := http.NewRequest("GET", "/Test/List", nil)
	w := httptest.NewRecorder()

	handler := NewControllerRegister()
	handler.AddAuto(&TestController{})
	handler.ServeHTTP(w, r)
	if w.Body.String() != "i am list" {
		t.Errorf("user define func can't run")
	}
}

func TestAutoFuncParams(t *testing.T) {
	r, _ := http.NewRequest("GET", "/test/params/2009/11/12", nil)
	w := httptest.NewRecorder()

	handler := NewControllerRegister()
	handler.AddAuto(&TestController{})
	handler.ServeHTTP(w, r)
	if w.Body.String() != "20091112" {
		t.Errorf("user define func can't run")
	}
}

func TestAutoExtFunc(t *testing.T) {
	r, _ := http.NewRequest("GET", "/test/myext.json", nil)
	w := httptest.NewRecorder()

	handler := NewControllerRegister()
	handler.AddAuto(&TestController{})
	handler.ServeHTTP(w, r)
	if w.Body.String() != "json" {
		t.Errorf("user define func can't run")
	}
}

func TestEscape(t *testing.T) {
	r, _ := http.NewRequest("GET", "/search/%E4%BD%A0%E5%A5%BD", nil)
	w := httptest.NewRecorder()

	handler := NewControllerRegister()
	handler.Get("/search/:keyword(.+)", func(ctx *context.Context) {
		value := ctx.Input.Param(":keyword")
		ctx.Output.Body([]byte(value))
	})
	handler.ServeHTTP(w, r)
	str := w.Body.String()
	if str != "你好" {
		t.Errorf("incorrect, %s", str)
	}
}

func TestRouteOk(t *testing.T) {
	r, _ := http.NewRequest("GET", "/person/anderson/thomas?learn=kungfu", nil)
	w := httptest.NewRecorder()

	handler := NewControllerRegister()
	handler.Add("/person/:last/:first", &TestController{}, WithRouterMethods(&TestController{}, "get:GetParams"))
	handler.ServeHTTP(w, r)
	body := w.Body.String()
	if body != "anderson+thomas+kungfu" {
		t.Errorf("url param set to [%s];", body)
	}
}

func TestManyRoute(t *testing.T) {
	r, _ := http.NewRequest("GET", "/beego32-12.html", nil)
	w := httptest.NewRecorder()

	handler := NewControllerRegister()
	handler.Add("/beego:id([0-9]+)-:page([0-9]+).html", &TestController{}, WithRouterMethods(&TestController{}, "get:GetManyRouter"))
	handler.ServeHTTP(w, r)

	body := w.Body.String()

	if body != "3212" {
		t.Errorf("url param set to [%s];", body)
	}
}

// Test for issue #1669
func TestEmptyResponse(t *testing.T) {
	r, _ := http.NewRequest("GET", "/beego-empty.html", nil)
	w := httptest.NewRecorder()

	handler := NewControllerRegister()
	handler.Add("/beego-empty.html", &TestController{}, WithRouterMethods(&TestController{}, "get:GetEmptyBody"))
	handler.ServeHTTP(w, r)

	if body := w.Body.String(); body != "" {
		t.Error("want empty body")
	}
}

func TestNotFound(t *testing.T) {
	r, _ := http.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	handler := NewControllerRegister()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("Code set to [%v]; want [%v]", w.Code, http.StatusNotFound)
	}
}

// TestStatic tests the ability to serve static
// content from the filesystem
func TestStatic(t *testing.T) {
	r, _ := http.NewRequest("GET", "/static/js/jquery.js", nil)
	w := httptest.NewRecorder()

	handler := NewControllerRegister()
	handler.ServeHTTP(w, r)

	if w.Code != 404 {
		t.Errorf("handler.Static failed to serve file")
	}
}

func TestPrepare(t *testing.T) {
	r, _ := http.NewRequest("GET", "/json/list", nil)
	w := httptest.NewRecorder()

	handler := NewControllerRegister()
	handler.Add("/json/list", &JSONController{})
	handler.ServeHTTP(w, r)
	if w.Body.String() != `"prepare"` {
		t.Errorf(w.Body.String() + "user define func can't run")
	}
}

func TestAutoPrefix(t *testing.T) {
	r, _ := http.NewRequest("GET", "/admin/test/list", nil)
	w := httptest.NewRecorder()

	handler := NewControllerRegister()
	handler.AddAutoPrefix("/admin", &TestController{})
	handler.ServeHTTP(w, r)
	if w.Body.String() != "i am list" {
		t.Errorf("TestAutoPrefix can't run")
	}
}

func TestCtrlGet(t *testing.T) {
	r, _ := http.NewRequest("GET", "/user", nil)
	w := httptest.NewRecorder()

	handler := NewControllerRegister()
	handler.Get("/user", func(ctx *context.Context) {
		ctx.Output.Body([]byte("Get userlist"))
	})
	handler.ServeHTTP(w, r)
	if w.Body.String() != "Get userlist" {
		t.Errorf("TestCtrlGet can't run")
	}
}

func TestCtrlPost(t *testing.T) {
	r, _ := http.NewRequest("POST", "/user/123", nil)
	w := httptest.NewRecorder()

	handler := NewControllerRegister()
	handler.Post("/user/:id", func(ctx *context.Context) {
		ctx.Output.Body([]byte(ctx.Input.Param(":id")))
	})
	handler.ServeHTTP(w, r)
	if w.Body.String() != "123" {
		t.Errorf("TestCtrlPost can't run")
	}
}

func sayhello(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("sayhello"))
}

func TestRouterHandler(t *testing.T) {
	r, _ := http.NewRequest("POST", "/sayhi", nil)
	w := httptest.NewRecorder()

	handler := NewControllerRegister()
	handler.Handler("/sayhi", http.HandlerFunc(sayhello))
	handler.ServeHTTP(w, r)
	if w.Body.String() != "sayhello" {
		t.Errorf("TestRouterHandler can't run")
	}
}

func TestRouterHandlerAll(t *testing.T) {
	r, _ := http.NewRequest("POST", "/sayhi/a/b/c", nil)
	w := httptest.NewRecorder()

	handler := NewControllerRegister()
	handler.Handler("/sayhi", http.HandlerFunc(sayhello), true)
	handler.ServeHTTP(w, r)
	if w.Body.String() != "sayhello" {
		t.Errorf("TestRouterHandler can't run")
	}
}

//
// Benchmarks NewHttpSever:
//

func beegoFilterFunc(ctx *context.Context) {
	ctx.WriteString("hello")
}

type AdminController struct {
	Controller
}

func (a *AdminController) Get() {
	a.Ctx.WriteString("hello")
}

func TestRouterFunc(t *testing.T) {
	mux := NewControllerRegister()
	mux.Get("/action", beegoFilterFunc)
	mux.Post("/action", beegoFilterFunc)
	rw, r := testRequest("GET", "/action")
	mux.ServeHTTP(rw, r)
	if rw.Body.String() != "hello" {
		t.Errorf("TestRouterFunc can't run")
	}
}

func BenchmarkFunc(b *testing.B) {
	mux := NewControllerRegister()
	mux.Get("/action", beegoFilterFunc)
	rw, r := testRequest("GET", "/action")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mux.ServeHTTP(rw, r)
	}
}

func BenchmarkController(b *testing.B) {
	mux := NewControllerRegister()
	mux.Add("/action", &AdminController{})
	rw, r := testRequest("GET", "/action")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mux.ServeHTTP(rw, r)
	}
}

func testRequest(method, path string) (*httptest.ResponseRecorder, *http.Request) {
	request, _ := http.NewRequest(method, path, nil)
	recorder := httptest.NewRecorder()

	return recorder, request
}

// Expectation: A Filter with the correct configuration should be created given
// specific parameters.
func TestInsertFilter(t *testing.T) {
	testName := "TestInsertFilter"

	mux := NewControllerRegister()
	mux.InsertFilter("*", BeforeRouter, func(*context.Context) {}, WithReturnOnOutput(true))
	if !mux.filters[BeforeRouter][0].returnOnOutput {
		t.Errorf(
			"%s: passing no variadic params should set returnOnOutput to true",
			testName)
	}
	if mux.filters[BeforeRouter][0].resetParams {
		t.Errorf(
			"%s: passing no variadic params should set resetParams to false",
			testName)
	}

	mux = NewControllerRegister()
	mux.InsertFilter("*", BeforeRouter, func(*context.Context) {}, WithReturnOnOutput(false))
	if mux.filters[BeforeRouter][0].returnOnOutput {
		t.Errorf(
			"%s: passing false as 1st variadic param should set returnOnOutput to false",
			testName)
	}

	mux = NewControllerRegister()
	mux.InsertFilter("*", BeforeRouter, func(*context.Context) {}, WithReturnOnOutput(true), WithResetParams(true))
	if !mux.filters[BeforeRouter][0].resetParams {
		t.Errorf(
			"%s: passing true as 2nd variadic param should set resetParams to true",
			testName)
	}
}

// Expectation: the second variadic arg should cause the execution of the filter
// to preserve the parameters from before its execution.
func TestParamResetFilter(t *testing.T) {
	testName := "TestParamResetFilter"
	route := "/beego/*" // splat
	path := "/beego/routes/routes"

	mux := NewControllerRegister()

	mux.InsertFilter("*", BeforeExec, beegoResetParams, WithReturnOnOutput(true), WithResetParams(true))

	mux.Get(route, beegoHandleResetParams)

	rw, r := testRequest("GET", path)
	mux.ServeHTTP(rw, r)

	// The two functions, `beegoResetParams` and `beegoHandleResetParams` add
	// a response header of `Splat`.  The expectation here is that Header
	// value should match what the _request's_ router set, not the filter's.

	headers := rw.Result().Header
	if len(headers["Splat"]) != 1 {
		t.Errorf(
			"%s: There was an error in the test. Splat param not set in Header",
			testName)
	}
	if headers["Splat"][0] != "routes/routes" {
		t.Errorf(
			"%s: expected `:splat` param to be [routes/routes] but it was [%s]",
			testName, headers["Splat"][0])
	}
}

// Execution point: BeforeRouter
// expectation: only BeforeRouter function is executed, notmatch output as router doesn't handle
func TestFilterBeforeRouter(t *testing.T) {
	testName := "TestFilterBeforeRouter"
	url := "/beforeRouter"

	mux := NewControllerRegister()
	mux.InsertFilter(url, BeforeRouter, beegoBeforeRouter1)

	mux.Get(url, beegoFilterFunc)

	rw, r := testRequest("GET", url)
	mux.ServeHTTP(rw, r)

	if !strings.Contains(rw.Body.String(), "BeforeRouter1") {
		t.Errorf(testName + " BeforeRouter did not run")
	}
	if strings.Contains(rw.Body.String(), "hello") {
		t.Errorf(testName + " BeforeRouter did not return properly")
	}
}

// Execution point: BeforeExec
// expectation: only BeforeExec function is executed, match as router determines route only
func TestFilterBeforeExec(t *testing.T) {
	testName := "TestFilterBeforeExec"
	url := "/beforeExec"

	mux := NewControllerRegister()
	mux.InsertFilter(url, BeforeRouter, beegoFilterNoOutput, WithReturnOnOutput(true))
	mux.InsertFilter(url, BeforeExec, beegoBeforeExec1, WithReturnOnOutput(true))

	mux.Get(url, beegoFilterFunc)

	rw, r := testRequest("GET", url)
	mux.ServeHTTP(rw, r)

	if !strings.Contains(rw.Body.String(), "BeforeExec1") {
		t.Errorf(testName + " BeforeExec did not run")
	}
	if strings.Contains(rw.Body.String(), "hello") {
		t.Errorf(testName + " BeforeExec did not return properly")
	}
	if strings.Contains(rw.Body.String(), "BeforeRouter") {
		t.Errorf(testName + " BeforeRouter ran in error")
	}
}

// Execution point: AfterExec
// expectation: only AfterExec function is executed, match as router handles
func TestFilterAfterExec(t *testing.T) {
	testName := "TestFilterAfterExec"
	url := "/afterExec"

	mux := NewControllerRegister()
	mux.InsertFilter(url, BeforeRouter, beegoFilterNoOutput)
	mux.InsertFilter(url, BeforeExec, beegoFilterNoOutput)
	mux.InsertFilter(url, AfterExec, beegoAfterExec1, WithReturnOnOutput(false))

	mux.Get(url, beegoFilterFunc)

	rw, r := testRequest("GET", url)
	mux.ServeHTTP(rw, r)

	if !strings.Contains(rw.Body.String(), "AfterExec1") {
		t.Errorf(testName + " AfterExec did not run")
	}
	if !strings.Contains(rw.Body.String(), "hello") {
		t.Errorf(testName + " handler did not run properly")
	}
	if strings.Contains(rw.Body.String(), "BeforeRouter") {
		t.Errorf(testName + " BeforeRouter ran in error")
	}
	if strings.Contains(rw.Body.String(), "BeforeExec") {
		t.Errorf(testName + " BeforeExec ran in error")
	}
}

// Execution point: FinishRouter
// expectation: only FinishRouter function is executed, match as router handles
func TestFilterFinishRouter(t *testing.T) {
	testName := "TestFilterFinishRouter"
	url := "/finishRouter"

	mux := NewControllerRegister()
	mux.InsertFilter(url, BeforeRouter, beegoFilterNoOutput, WithReturnOnOutput(true))
	mux.InsertFilter(url, BeforeExec, beegoFilterNoOutput, WithReturnOnOutput(true))
	mux.InsertFilter(url, AfterExec, beegoFilterNoOutput, WithReturnOnOutput(true))
	mux.InsertFilter(url, FinishRouter, beegoFinishRouter1, WithReturnOnOutput(true))

	mux.Get(url, beegoFilterFunc)

	rw, r := testRequest("GET", url)
	mux.ServeHTTP(rw, r)

	if strings.Contains(rw.Body.String(), "FinishRouter1") {
		t.Errorf(testName + " FinishRouter did not run")
	}
	if !strings.Contains(rw.Body.String(), "hello") {
		t.Errorf(testName + " handler did not run properly")
	}
	if strings.Contains(rw.Body.String(), "AfterExec1") {
		t.Errorf(testName + " AfterExec ran in error")
	}
	if strings.Contains(rw.Body.String(), "BeforeRouter") {
		t.Errorf(testName + " BeforeRouter ran in error")
	}
	if strings.Contains(rw.Body.String(), "BeforeExec") {
		t.Errorf(testName + " BeforeExec ran in error")
	}
}

// Execution point: FinishRouter
// expectation: only first FinishRouter function is executed, match as router handles
func TestFilterFinishRouterMultiFirstOnly(t *testing.T) {
	testName := "TestFilterFinishRouterMultiFirstOnly"
	url := "/finishRouterMultiFirstOnly"

	mux := NewControllerRegister()
	mux.InsertFilter(url, FinishRouter, beegoFinishRouter1, WithReturnOnOutput(false))
	mux.InsertFilter(url, FinishRouter, beegoFinishRouter2)

	mux.Get(url, beegoFilterFunc)

	rw, r := testRequest("GET", url)
	mux.ServeHTTP(rw, r)

	if !strings.Contains(rw.Body.String(), "FinishRouter1") {
		t.Errorf(testName + " FinishRouter1 did not run")
	}
	if !strings.Contains(rw.Body.String(), "hello") {
		t.Errorf(testName + " handler did not run properly")
	}
	// not expected in body
	if strings.Contains(rw.Body.String(), "FinishRouter2") {
		t.Errorf(testName + " FinishRouter2 did run")
	}
}

// Execution point: FinishRouter
// expectation: both FinishRouter functions execute, match as router handles
func TestFilterFinishRouterMulti(t *testing.T) {
	testName := "TestFilterFinishRouterMulti"
	url := "/finishRouterMulti"

	mux := NewControllerRegister()
	mux.InsertFilter(url, FinishRouter, beegoFinishRouter1, WithReturnOnOutput(false))
	mux.InsertFilter(url, FinishRouter, beegoFinishRouter2, WithReturnOnOutput(false))

	mux.Get(url, beegoFilterFunc)

	rw, r := testRequest("GET", url)
	mux.ServeHTTP(rw, r)

	if !strings.Contains(rw.Body.String(), "FinishRouter1") {
		t.Errorf(testName + " FinishRouter1 did not run")
	}
	if !strings.Contains(rw.Body.String(), "hello") {
		t.Errorf(testName + " handler did not run properly")
	}
	if !strings.Contains(rw.Body.String(), "FinishRouter2") {
		t.Errorf(testName + " FinishRouter2 did not run properly")
	}
}

func beegoFilterNoOutput(ctx *context.Context) {
}

func beegoBeforeRouter1(ctx *context.Context) {
	ctx.WriteString("|BeforeRouter1")
}

func beegoBeforeExec1(ctx *context.Context) {
	ctx.WriteString("|BeforeExec1")
}

func beegoAfterExec1(ctx *context.Context) {
	ctx.WriteString("|AfterExec1")
}

func beegoFinishRouter1(ctx *context.Context) {
	ctx.WriteString("|FinishRouter1")
}

func beegoFinishRouter2(ctx *context.Context) {
	ctx.WriteString("|FinishRouter2")
}

func beegoResetParams(ctx *context.Context) {
	ctx.ResponseWriter.Header().Set("splat", ctx.Input.Param(":splat"))
}

func beegoHandleResetParams(ctx *context.Context) {
	ctx.ResponseWriter.Header().Set("splat", ctx.Input.Param(":splat"))
}

// YAML
type YAMLController struct {
	Controller
}

func (jc *YAMLController) Prepare() {
	jc.Data["yaml"] = "prepare"
	jc.ServeYAML()
}

func (jc *YAMLController) Get() {
	jc.Data["Username"] = "astaxie"
	jc.Ctx.Output.Body([]byte("ok"))
}

func TestYAMLPrepare(t *testing.T) {
	r, _ := http.NewRequest("GET", "/yaml/list", nil)
	w := httptest.NewRecorder()

	handler := NewControllerRegister()
	handler.Add("/yaml/list", &YAMLController{})
	handler.ServeHTTP(w, r)
	if strings.TrimSpace(w.Body.String()) != "prepare" {
		t.Errorf(w.Body.String())
	}
}

func TestRouterEntityTooLargeCopyBody(t *testing.T) {
	_MaxMemory := BConfig.MaxMemory
	_CopyRequestBody := BConfig.CopyRequestBody
	BConfig.CopyRequestBody = true
	BConfig.MaxMemory = 20

	BeeApp.Cfg.CopyRequestBody = true
	BeeApp.Cfg.MaxMemory = 20
	b := bytes.NewBuffer([]byte("barbarbarbarbarbarbarbarbarbar"))
	r, _ := http.NewRequest("POST", "/user/123", b)
	w := httptest.NewRecorder()

	handler := NewControllerRegister()
	handler.Post("/user/:id", func(ctx *context.Context) {
		ctx.Output.Body([]byte(ctx.Input.Param(":id")))
	})
	handler.ServeHTTP(w, r)

	BConfig.CopyRequestBody = _CopyRequestBody
	BConfig.MaxMemory = _MaxMemory

	if w.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("TestRouterRequestEntityTooLarge can't run")
	}
}

func TestRouterSessionSet(t *testing.T) {
	oldGlobalSessionOn := BConfig.WebConfig.Session.SessionOn
	defer func() {
		BConfig.WebConfig.Session.SessionOn = oldGlobalSessionOn
	}()

	// global sessionOn = false, router sessionOn = false
	r, _ := http.NewRequest("GET", "/user", nil)
	w := httptest.NewRecorder()
	handler := NewControllerRegister()
	handler.Add("/user", &TestController{}, WithRouterMethods(&TestController{}, "get:Get"),
		WithRouterSessionOn(false))
	handler.ServeHTTP(w, r)
	if w.Header().Get("Set-Cookie") != "" {
		t.Errorf("TestRotuerSessionSet failed")
	}

	// global sessionOn = false, router sessionOn = true
	r, _ = http.NewRequest("GET", "/user", nil)
	w = httptest.NewRecorder()
	handler = NewControllerRegister()
	handler.Add("/user", &TestController{}, WithRouterMethods(&TestController{}, "get:Get"),
		WithRouterSessionOn(true))
	handler.ServeHTTP(w, r)
	if w.Header().Get("Set-Cookie") != "" {
		t.Errorf("TestRotuerSessionSet failed")
	}

	BConfig.WebConfig.Session.SessionOn = true
	if err := registerSession(); err != nil {
		t.Errorf("register session failed, error: %s", err.Error())
	}
	// global sessionOn = true, router sessionOn = false
	r, _ = http.NewRequest("GET", "/user", nil)
	w = httptest.NewRecorder()
	handler = NewControllerRegister()
	handler.Add("/user", &TestController{}, WithRouterMethods(&TestController{}, "get:Get"),
		WithRouterSessionOn(false))
	handler.ServeHTTP(w, r)
	if w.Header().Get("Set-Cookie") != "" {
		t.Errorf("TestRotuerSessionSet failed")
	}

	// global sessionOn = true, router sessionOn = true
	r, _ = http.NewRequest("GET", "/user", nil)
	w = httptest.NewRecorder()
	handler = NewControllerRegister()
	handler.Add("/user", &TestController{}, WithRouterMethods(&TestController{}, "get:Get"),
		WithRouterSessionOn(true))
	handler.ServeHTTP(w, r)
	if w.Header().Get("Set-Cookie") == "" {
		t.Errorf("TestRotuerSessionSet failed")
	}
}

func TestRouterCtrlGet(t *testing.T) {
	r, _ := http.NewRequest(http.MethodGet, "/user", nil)
	w := httptest.NewRecorder()

	handler := NewControllerRegister()
	handler.CtrlGet("/user", ExampleController.Ping)
	handler.ServeHTTP(w, r)
	if w.Body.String() != exampleBody {
		t.Errorf("TestRouterCtrlGet can't run")
	}
}

func TestRouterCtrlPost(t *testing.T) {
	r, _ := http.NewRequest(http.MethodPost, "/user", nil)
	w := httptest.NewRecorder()

	handler := NewControllerRegister()
	handler.CtrlPost("/user", ExampleController.Ping)
	handler.ServeHTTP(w, r)
	if w.Body.String() != exampleBody {
		t.Errorf("TestRouterCtrlPost can't run")
	}
}

func TestRouterCtrlHead(t *testing.T) {
	r, _ := http.NewRequest(http.MethodHead, "/user", nil)
	w := httptest.NewRecorder()

	handler := NewControllerRegister()
	handler.CtrlHead("/user", ExampleController.Ping)
	handler.ServeHTTP(w, r)
	if w.Body.String() != exampleBody {
		t.Errorf("TestRouterCtrlHead can't run")
	}
}

func TestRouterCtrlPut(t *testing.T) {
	r, _ := http.NewRequest(http.MethodPut, "/user", nil)
	w := httptest.NewRecorder()

	handler := NewControllerRegister()
	handler.CtrlPut("/user", ExampleController.Ping)
	handler.ServeHTTP(w, r)
	if w.Body.String() != exampleBody {
		t.Errorf("TestRouterCtrlPut can't run")
	}
}

func TestRouterCtrlPatch(t *testing.T) {
	r, _ := http.NewRequest(http.MethodPatch, "/user", nil)
	w := httptest.NewRecorder()

	handler := NewControllerRegister()
	handler.CtrlPatch("/user", ExampleController.Ping)
	handler.ServeHTTP(w, r)
	if w.Body.String() != exampleBody {
		t.Errorf("TestRouterCtrlPatch can't run")
	}
}

func TestRouterCtrlDelete(t *testing.T) {
	r, _ := http.NewRequest(http.MethodDelete, "/user", nil)
	w := httptest.NewRecorder()

	handler := NewControllerRegister()
	handler.CtrlDelete("/user", ExampleController.Ping)
	handler.ServeHTTP(w, r)
	if w.Body.String() != exampleBody {
		t.Errorf("TestRouterCtrlDelete can't run")
	}
}

func TestRouterCtrlAny(t *testing.T) {
	handler := NewControllerRegister()
	handler.CtrlAny("/user", ExampleController.Ping)

	for method := range HTTPMETHOD {
		w := httptest.NewRecorder()
		r, _ := http.NewRequest(method, "/user", nil)
		handler.ServeHTTP(w, r)
		if w.Body.String() != exampleBody {
			t.Errorf("TestRouterCtrlAny can't run, get the response is " + w.Body.String())
		}
	}
}

func TestRouterCtrlGetPointerMethod(t *testing.T) {
	r, _ := http.NewRequest(http.MethodGet, "/user", nil)
	w := httptest.NewRecorder()

	handler := NewControllerRegister()
	handler.CtrlGet("/user", (*ExampleController).PingPointer)
	handler.ServeHTTP(w, r)
	if w.Body.String() != examplePointerBody {
		t.Errorf("TestRouterCtrlGetPointerMethod can't run")
	}
}

func TestRouterCtrlPostPointerMethod(t *testing.T) {
	r, _ := http.NewRequest(http.MethodPost, "/user", nil)
	w := httptest.NewRecorder()

	handler := NewControllerRegister()
	handler.CtrlPost("/user", (*ExampleController).PingPointer)
	handler.ServeHTTP(w, r)
	if w.Body.String() != examplePointerBody {
		t.Errorf("TestRouterCtrlPostPointerMethod can't run")
	}
}

func TestRouterCtrlHeadPointerMethod(t *testing.T) {
	r, _ := http.NewRequest(http.MethodHead, "/user", nil)
	w := httptest.NewRecorder()

	handler := NewControllerRegister()
	handler.CtrlHead("/user", (*ExampleController).PingPointer)
	handler.ServeHTTP(w, r)
	if w.Body.String() != examplePointerBody {
		t.Errorf("TestRouterCtrlHeadPointerMethod can't run")
	}
}

func TestRouterCtrlPutPointerMethod(t *testing.T) {
	r, _ := http.NewRequest(http.MethodPut, "/user", nil)
	w := httptest.NewRecorder()

	handler := NewControllerRegister()
	handler.CtrlPut("/user", (*ExampleController).PingPointer)
	handler.ServeHTTP(w, r)
	if w.Body.String() != examplePointerBody {
		t.Errorf("TestRouterCtrlPutPointerMethod can't run")
	}
}

func TestRouterCtrlPatchPointerMethod(t *testing.T) {
	r, _ := http.NewRequest(http.MethodPatch, "/user", nil)
	w := httptest.NewRecorder()

	handler := NewControllerRegister()
	handler.CtrlPatch("/user", (*ExampleController).PingPointer)
	handler.ServeHTTP(w, r)
	if w.Body.String() != examplePointerBody {
		t.Errorf("TestRouterCtrlPatchPointerMethod can't run")
	}
}

func TestRouterCtrlDeletePointerMethod(t *testing.T) {
	r, _ := http.NewRequest(http.MethodDelete, "/user", nil)
	w := httptest.NewRecorder()

	handler := NewControllerRegister()
	handler.CtrlDelete("/user", (*ExampleController).PingPointer)
	handler.ServeHTTP(w, r)
	if w.Body.String() != examplePointerBody {
		t.Errorf("TestRouterCtrlDeletePointerMethod can't run")
	}
}

func TestRouterCtrlAnyPointerMethod(t *testing.T) {
	handler := NewControllerRegister()
	handler.CtrlAny("/user", (*ExampleController).PingPointer)

	for method := range HTTPMETHOD {
		w := httptest.NewRecorder()
		r, _ := http.NewRequest(method, "/user", nil)
		handler.ServeHTTP(w, r)
		if w.Body.String() != examplePointerBody {
			t.Errorf("TestRouterCtrlAnyPointerMethod can't run, get the response is " + w.Body.String())
		}
	}
}

func TestRouterAddRouterMethodPanicInvalidMethod(t *testing.T) {
	method := "some random method"
	message := "not support http method: " + strings.ToUpper(method)
	defer func() {
		err := recover()
		if err != nil { // 产生了panic异常
			errStr, ok := err.(string)
			if ok && errStr == message {
				return
			}
		}
		t.Errorf(fmt.Sprintf("TestRouterAddRouterMethodPanicInvalidMethod failed: %v", err))
	}()

	handler := NewControllerRegister()
	handler.AddRouterMethod(method, "/user", ExampleController.Ping)
}

func TestRouterAddRouterMethodPanicNotAMethod(t *testing.T) {
	method := http.MethodGet
	message := "not a method"
	defer func() {
		err := recover()
		if err != nil { // 产生了panic异常
			errStr, ok := err.(string)
			if ok && errStr == message {
				return
			}
		}
		t.Errorf(fmt.Sprintf("TestRouterAddRouterMethodPanicNotAMethod failed: %v", err))
	}()

	handler := NewControllerRegister()
	handler.AddRouterMethod(method, "/user", ExampleController{})
}

func TestRouterAddRouterMethodPanicNotPublicMethod(t *testing.T) {
	method := http.MethodGet
	message := "ping is not a public method"
	defer func() {
		err := recover()
		if err != nil { // 产生了panic异常
			errStr, ok := err.(string)
			if ok && errStr == message {
				return
			}
		}
		t.Errorf(fmt.Sprintf("TestRouterAddRouterMethodPanicNotPublicMethod failed: %v", err))
	}()

	handler := NewControllerRegister()
	handler.AddRouterMethod(method, "/user", ExampleController.ping)
}

func TestRouterAddRouterMethodPanicNotImplementInterface(t *testing.T) {
	method := http.MethodGet
	message := "web.TestControllerWithInterface is not implemented ControllerInterface"
	defer func() {
		err := recover()
		if err != nil { // 产生了panic异常
			errStr, ok := err.(string)
			if ok && errStr == message {
				return
			}
		}
		t.Errorf(fmt.Sprintf("TestRouterAddRouterMethodPanicNotImplementInterface failed: %v", err))
	}()

	handler := NewControllerRegister()
	handler.AddRouterMethod(method, "/user", TestControllerWithInterface.Ping)
}

func TestRouterAddRouterPointerMethodPanicNotImplementInterface(t *testing.T) {
	method := http.MethodGet
	message := "web.TestControllerWithInterface is not implemented ControllerInterface"
	defer func() {
		err := recover()
		if err != nil { // 产生了panic异常
			errStr, ok := err.(string)
			if ok && errStr == message {
				return
			}
		}
		t.Errorf(fmt.Sprintf("TestRouterAddRouterPointerMethodPanicNotImplementInterface failed: %v", err))
	}()

	handler := NewControllerRegister()
	handler.AddRouterMethod(method, "/user", (*TestControllerWithInterface).PingPointer)
}

func TestGetAllControllerInfo(t *testing.T) {
	handler := NewControllerRegister()
	handler.Add("/level1", &TestController{}, WithRouterMethods(&TestController{}, "get:Get"))
	handler.Add("/level1/level2", &TestController{}, WithRouterMethods(&TestController{}, "get:Get"))
	handler.Add("/:name1", &TestController{}, WithRouterMethods(&TestController{}, "post:Post"))

	var actualPatterns []string
	var actualMethods []string
	for _, controllerInfo := range handler.GetAllControllerInfo() {
		actualPatterns = append(actualPatterns, controllerInfo.GetPattern())
		for _, httpMethod := range controllerInfo.GetMethod() {
			actualMethods = append(actualMethods, httpMethod)
		}
	}
	sort.Strings(actualPatterns)
	expectedPatterns := []string{"/level1", "/level1/level2", "/:name1"}
	sort.Strings(expectedPatterns)
	if !reflect.DeepEqual(actualPatterns, expectedPatterns) {
		t.Errorf("ControllerInfo.GetMethod expected %#v, but %#v got", expectedPatterns, actualPatterns)
	}

	sort.Strings(actualMethods)
	expectedMethods := []string{"Get", "Get", "Post"}
	sort.Strings(expectedMethods)
	if !reflect.DeepEqual(actualMethods, expectedMethods) {
		t.Errorf("ControllerInfo.GetMethod expected %#v, but %#v got", expectedMethods, actualMethods)
	}
}
