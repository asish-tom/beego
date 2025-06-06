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
	"net/http"
	"os"
	"path/filepath"
	"testing"

	assetfs "github.com/elazarl/go-bindata-assetfs"
	"github.com/stretchr/testify/assert"

	"github.com/asish-tom/beego/v2/test"
)

var header = `{{define "header"}}
<h1>Hello, astaxie!</h1>
{{end}}`

var index = `<!DOCTYPE html>
<html>
  <head>
    <title>beego welcome template</title>
  </head>
  <body>
{{template "block"}}
{{template "header"}}
{{template "blocks/block.tpl"}}
  </body>
</html>
`

var block = `{{define "block"}}
<h1>Hello, blocks!</h1>
{{end}}`

func TestTemplate(t *testing.T) {
	tmpDir := os.TempDir()
	dir := filepath.Join(tmpDir, "_beeTmp", "TestTemplate")
	files := []string{
		"header.tpl",
		"index.tpl",
		"blocks/block.tpl",
	}
	if err := os.MkdirAll(dir, 0o777); err != nil {
		t.Fatal(err)
	}
	for k, name := range files {
		dirErr := os.MkdirAll(filepath.Dir(filepath.Join(dir, name)), 0o777)
		assert.Nil(t, dirErr)
		if f, err := os.Create(filepath.Join(dir, name)); err != nil {
			t.Fatal(err)
		} else {
			if k == 0 {
				f.WriteString(header)
			} else if k == 1 {
				f.WriteString(index)
			} else if k == 2 {
				f.WriteString(block)
			}

			f.Close()
		}
	}
	if err := AddViewPath(dir); err != nil {
		t.Fatal(err)
	}
	beeTemplates := beeViewPathTemplates[dir]
	if len(beeTemplates) != 3 {
		t.Fatalf("should be 3 but got %v", len(beeTemplates))
	}
	if err := beeTemplates["index.tpl"].ExecuteTemplate(os.Stdout, "index.tpl", nil); err != nil {
		t.Fatal(err)
	}
	for _, name := range files {
		os.RemoveAll(filepath.Join(dir, name))
	}
	os.RemoveAll(dir)
}

var menu = `<div class="menu">
<ul>
<li>menu1</li>
<li>menu2</li>
<li>menu3</li>
</ul>
</div>
`

var user = `<!DOCTYPE html>
<html>
  <head>
    <title>beego welcome template</title>
  </head>
  <body>
{{template "../public/menu.tpl"}}
  </body>
</html>
`

func TestRelativeTemplate(t *testing.T) {
	tmpDir := os.TempDir()
	dir := filepath.Join(tmpDir, "_beeTmp")

	// Just add dir to known viewPaths
	if err := AddViewPath(dir); err != nil {
		t.Fatal(err)
	}

	files := []string{
		"easyui/public/menu.tpl",
		"easyui/rbac/user.tpl",
	}
	if err := os.MkdirAll(dir, 0o777); err != nil {
		t.Fatal(err)
	}
	for k, name := range files {
		os.MkdirAll(filepath.Dir(filepath.Join(dir, name)), 0o777)
		if f, err := os.Create(filepath.Join(dir, name)); err != nil {
			t.Fatal(err)
		} else {
			if k == 0 {
				f.WriteString(menu)
			} else if k == 1 {
				f.WriteString(user)
			}
			f.Close()
		}
	}
	if err := BuildTemplate(dir, files[1]); err != nil {
		t.Fatal(err)
	}
	beeTemplates := beeViewPathTemplates[dir]
	if err := beeTemplates["easyui/rbac/user.tpl"].ExecuteTemplate(os.Stdout, "easyui/rbac/user.tpl", nil); err != nil {
		t.Fatal(err)
	}
	for _, name := range files {
		os.RemoveAll(filepath.Join(dir, name))
	}
	os.RemoveAll(dir)
}

var add = `{{ template "layout_blog.tpl" . }}
{{ define "css" }}
        <link rel="stylesheet" href="/static/css/current.css">
{{ end}}


{{ define "content" }}
        <h2>{{ .Title }}</h2>
        <p> This is SomeVar: {{ .SomeVar }}</p>
{{ end }}

{{ define "js" }}
    <script src="/static/js/current.js"></script>
{{ end}}`

var layoutBlog = `<!DOCTYPE html>
<html>
<head>
    <title>Lin Li</title>
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <meta http-equiv="Content-Type" content="text/html; charset=utf-8">
    <link rel="stylesheet" href="http://netdna.bootstrapcdn.com/bootstrap/3.0.3/css/bootstrap.min.css">
    <link rel="stylesheet" href="http://netdna.bootstrapcdn.com/bootstrap/3.0.3/css/bootstrap-theme.min.css">
     {{ block "css" . }}{{ end }}
</head>
<body>

    <div class="container">
        {{ block "content" . }}{{ end }}
    </div>
    <script type="text/javascript" src="http://code.jquery.com/jquery-2.0.3.min.js"></script>
    <script src="http://netdna.bootstrapcdn.com/bootstrap/3.0.3/js/bootstrap.min.js"></script>
     {{ block "js" . }}{{ end }}
</body>
</html>`

var output = `<!DOCTYPE html>
<html>
<head>
    <title>Lin Li</title>
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <meta http-equiv="Content-Type" content="text/html; charset=utf-8">
    <link rel="stylesheet" href="http://netdna.bootstrapcdn.com/bootstrap/3.0.3/css/bootstrap.min.css">
    <link rel="stylesheet" href="http://netdna.bootstrapcdn.com/bootstrap/3.0.3/css/bootstrap-theme.min.css">
     
        <link rel="stylesheet" href="/static/css/current.css">

</head>
<body>

    <div class="container">
        
        <h2>Hello</h2>
        <p> This is SomeVar: val</p>

    </div>
    <script type="text/javascript" src="http://code.jquery.com/jquery-2.0.3.min.js"></script>
    <script src="http://netdna.bootstrapcdn.com/bootstrap/3.0.3/js/bootstrap.min.js"></script>
     
    <script src="/static/js/current.js"></script>

</body>
</html>





`

func TestTemplateLayout(t *testing.T) {
	tmpDir, err := os.Getwd()
	assert.Nil(t, err)

	dir := filepath.Join(tmpDir, "_beeTmp", "TestTemplateLayout")
	files := []string{
		"add.tpl",
		"layout_blog.tpl",
	}
	if err := os.MkdirAll(dir, 0o777); err != nil {
		t.Fatal(err)
	}

	for k, name := range files {
		dirErr := os.MkdirAll(filepath.Dir(filepath.Join(dir, name)), 0o777)
		assert.Nil(t, dirErr)
		if f, err := os.Create(filepath.Join(dir, name)); err != nil {
			t.Fatal(err)
		} else {
			if k == 0 {
				_, writeErr := f.WriteString(add)
				assert.Nil(t, writeErr)
			} else if k == 1 {
				_, writeErr := f.WriteString(layoutBlog)
				assert.Nil(t, writeErr)
			}
			clErr := f.Close()
			assert.Nil(t, clErr)
		}
	}
	if err := AddViewPath(dir); err != nil {
		t.Fatal(err)
	}
	beeTemplates := beeViewPathTemplates[dir]
	if len(beeTemplates) != 2 {
		t.Fatalf("should be 2 but got %v", len(beeTemplates))
	}
	out := bytes.NewBufferString("")

	if err := beeTemplates["add.tpl"].ExecuteTemplate(out, "add.tpl", map[string]string{"Title": "Hello", "SomeVar": "val"}); err != nil {
		t.Fatal(err)
	}
	if out.String() != output {
		t.Log(out.String())
		t.Fatal("Compare failed")
	}
	for _, name := range files {
		os.RemoveAll(filepath.Join(dir, name))
	}
	os.RemoveAll(dir)
}

type TestingFileSystem struct {
	assetfs *assetfs.AssetFS
}

func (d TestingFileSystem) Open(name string) (http.File, error) {
	return d.assetfs.Open(name)
}

var outputBinData = `<!DOCTYPE html>
<html>
  <head>
    <title>beego welcome template</title>
  </head>
  <body>

	
<h1>Hello, blocks!</h1>

	
<h1>Hello, astaxie!</h1>

	

	<h2>Hello</h2>
	<p> This is SomeVar: val</p>
  </body>
</html>
`

func TestFsBinData(t *testing.T) {
	SetTemplateFSFunc(func() http.FileSystem {
		return TestingFileSystem{&assetfs.AssetFS{Asset: test.Asset, AssetDir: test.AssetDir, AssetInfo: test.AssetInfo}}
	})
	dir := "views"
	if err := AddViewPath("views"); err != nil {
		t.Fatal(err)
	}
	beeTemplates := beeViewPathTemplates[dir]
	if len(beeTemplates) != 3 {
		t.Fatalf("should be 3 but got %v", len(beeTemplates))
	}
	if err := beeTemplates["index.tpl"].ExecuteTemplate(os.Stdout, "index.tpl", map[string]string{"Title": "Hello", "SomeVar": "val"}); err != nil {
		t.Fatal(err)
	}
	out := bytes.NewBufferString("")
	if err := beeTemplates["index.tpl"].ExecuteTemplate(out, "index.tpl", map[string]string{"Title": "Hello", "SomeVar": "val"}); err != nil {
		t.Fatal(err)
	}

	if out.String() != outputBinData {
		t.Log(out.String())
		t.Fatal("Compare failed")
	}
}
