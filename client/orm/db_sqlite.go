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

package orm

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/asish-tom/beego/v2/client/orm/internal/models"

	"github.com/asish-tom/beego/v2/client/orm/hints"
)

// sqlite operators.
var sqliteOperators = map[string]string{
	"exact":       "= ?",
	"iexact":      "LIKE ? ESCAPE '\\'",
	"contains":    "LIKE ? ESCAPE '\\'",
	"icontains":   "LIKE ? ESCAPE '\\'",
	"gt":          "> ?",
	"gte":         ">= ?",
	"lt":          "< ?",
	"lte":         "<= ?",
	"eq":          "= ?",
	"ne":          "!= ?",
	"startswith":  "LIKE ? ESCAPE '\\'",
	"endswith":    "LIKE ? ESCAPE '\\'",
	"istartswith": "LIKE ? ESCAPE '\\'",
	"iendswith":   "LIKE ? ESCAPE '\\'",
}

// sqlite column types.
var sqliteTypes = map[string]string{
	"auto":                "integer NOT NULL PRIMARY KEY AUTOINCREMENT",
	"pk":                  "NOT NULL PRIMARY KEY",
	"bool":                "bool",
	"string":              "varchar(%d)",
	"string-char":         "character(%d)",
	"string-text":         "text",
	"time.Time-date":      "date",
	"time.Time":           "datetime",
	"time.Time-precision": "datetime(%d)",
	"int8":                "tinyint",
	"int16":               "smallint",
	"int32":               "integer",
	"int64":               "bigint",
	"uint8":               "tinyint unsigned",
	"uint16":              "smallint unsigned",
	"uint32":              "integer unsigned",
	"uint64":              "bigint unsigned",
	"float64":             "real",
	"float64-decimal":     "decimal",
}

// sqlite dbBaser.
type dbBaseSqlite struct {
	dbBase
}

var _ dbBaser = new(dbBaseSqlite)

// override base db read for update behavior as SQlite does not support syntax
func (d *dbBaseSqlite) Read(ctx context.Context, q dbQuerier, mi *models.ModelInfo, ind reflect.Value, tz *time.Location, cols []string, isForUpdate bool) error {
	if isForUpdate {
		DebugLog.Println("[WARN] SQLite does not support SELECT FOR UPDATE query, isForUpdate param is ignored and always as false to do the work")
	}
	return d.dbBase.Read(ctx, q, mi, ind, tz, cols, false)
}

// Get sqlite operator.
func (d *dbBaseSqlite) OperatorSQL(operator string) string {
	return sqliteOperators[operator]
}

// generate functioned sql for sqlite.
// only support DATE(text).
func (d *dbBaseSqlite) GenerateOperatorLeftCol(fi *models.FieldInfo, operator string, leftCol *string) {
	if fi.FieldType == TypeDateField {
		*leftCol = fmt.Sprintf("DATE(%s)", *leftCol)
	}
}

// unable updating joined record in sqlite.
func (d *dbBaseSqlite) SupportUpdateJoin() bool {
	return false
}

// max int in sqlite.
func (d *dbBaseSqlite) MaxLimit() uint64 {
	return 9223372036854775807
}

// Get column types in sqlite.
func (d *dbBaseSqlite) DbTypes() map[string]string {
	return sqliteTypes
}

// Get show tables sql in sqlite.
func (d *dbBaseSqlite) ShowTablesQuery() string {
	return "SELECT name FROM sqlite_master WHERE type = 'table'"
}

// Get Columns in sqlite.
func (d *dbBaseSqlite) GetColumns(ctx context.Context, db dbQuerier, table string) (map[string][3]string, error) {
	query := d.ins.ShowColumnsQuery(table)

	fullQuery := query // Default to original query
	// Check if the dbQuerier supports getting comments (might be raw *sql.DB during syncdb)
	if qcGetter, ok := db.(interface {
		GetQueryComments() *QueryComments
	}); ok {
		qc := qcGetter.GetQueryComments()
		if qc != nil {
			commentStr := qc.String()
			fullQuery = commentStr + query // Prepend only if supported and non-nil
		}
	}

	rows, err := db.QueryContext(ctx, fullQuery)
	if err != nil {
		return nil, err
	}

	columns := make(map[string][3]string)
	for rows.Next() {
		var tmp, name, typ, null sql.NullString
		err := rows.Scan(&tmp, &name, &typ, &null, &tmp, &tmp)
		if err != nil {
			return nil, err
		}
		columns[name.String] = [3]string{name.String, typ.String, null.String}
	}

	return columns, rows.Err()
}

// Get show Columns sql in sqlite.
func (d *dbBaseSqlite) ShowColumnsQuery(table string) string {
	return fmt.Sprintf("pragma table_info('%s')", table)
}

// check index exist in sqlite.
func (d *dbBaseSqlite) IndexExists(ctx context.Context, db dbQuerier, table string, name string) bool {
	query := fmt.Sprintf("PRAGMA index_list('%s')", table)

	fullQuery := query // Default to original query
	// Check if the dbQuerier supports getting comments (might be raw *sql.DB during syncdb)
	if qcGetter, ok := db.(interface {
		GetQueryComments() *QueryComments
	}); ok {
		qc := qcGetter.GetQueryComments()
		if qc != nil {
			commentStr := qc.String()
			fullQuery = commentStr + query // Prepend only if supported and non-nil
		}
	}

	rows, err := db.QueryContext(ctx, fullQuery)
	if err != nil {
		// Consider logging or returning the error instead of panicking
		// For now, keeping the panic to match original behavior on error
		panic(err)
	}
	defer rows.Close()
	for rows.Next() {
		var tmp, index sql.NullString
		rows.Scan(&tmp, &index, &tmp, &tmp, &tmp)
		if name == index.String {
			return true
		}
	}
	return false
}

// GenerateSpecifyIndex return a specifying index clause
func (d *dbBaseSqlite) GenerateSpecifyIndex(tableName string, useIndex int, indexes []string) string {
	var s []string
	Q := d.TableQuote()
	for _, index := range indexes {
		tmp := fmt.Sprintf(`%s%s%s`, Q, index, Q)
		s = append(s, tmp)
	}

	switch useIndex {
	case hints.KeyUseIndex, hints.KeyForceIndex:
		return fmt.Sprintf(` INDEXED BY %s `, strings.Join(s, `,`))
	default:
		DebugLog.Println("[WARN] Not a valid specifying action, so that action is ignored")
		return ``
	}
}

// Helper method to handle comment prepending with type assertion
func (d *dbBaseSqlite) prependCommentsIfSupported(db dbQuerier, query string) string {
	// Check if the dbQuerier supports getting comments (might be raw *sql.DB during syncdb)
	if qcGetter, ok := db.(interface {
		GetQueryComments() *QueryComments
	}); ok {
		if qc := qcGetter.GetQueryComments(); qc != nil {
			return qc.String() + query
		}
	}
	return query
}

// create new sqlite dbBaser.
func newdbBaseSqlite() dbBaser {
	b := new(dbBaseSqlite)
	b.ins = b
	return b
}
