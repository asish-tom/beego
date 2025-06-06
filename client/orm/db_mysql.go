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
	"fmt"
	"reflect"
	"strings"

	"github.com/asish-tom/beego/v2/client/orm/internal/models"
)

// mysql operators.
var mysqlOperators = map[string]string{
	"exact":       "= ?",
	"iexact":      "LIKE ?",
	"strictexact": "= BINARY ?",
	"contains":    "LIKE BINARY ?",
	"icontains":   "LIKE ?",
	// "regex":       "REGEXP BINARY ?",
	// "iregex":      "REGEXP ?",
	"gt":          "> ?",
	"gte":         ">= ?",
	"lt":          "< ?",
	"lte":         "<= ?",
	"eq":          "= ?",
	"ne":          "!= ?",
	"startswith":  "LIKE BINARY ?",
	"endswith":    "LIKE BINARY ?",
	"istartswith": "LIKE ?",
	"iendswith":   "LIKE ?",
}

// mysql column field types.
var mysqlTypes = map[string]string{
	"auto":                "AUTO_INCREMENT NOT NULL PRIMARY KEY",
	"pk":                  "NOT NULL PRIMARY KEY",
	"bool":                "bool",
	"string":              "varchar(%d)",
	"string-char":         "char(%d)",
	"string-text":         "longtext",
	"time.Time-date":      "date",
	"time.Time":           "datetime",
	"int8":                "tinyint",
	"int16":               "smallint",
	"int32":               "integer",
	"int64":               "bigint",
	"uint8":               "tinyint unsigned",
	"uint16":              "smallint unsigned",
	"uint32":              "integer unsigned",
	"uint64":              "bigint unsigned",
	"float64":             "double precision",
	"float64-decimal":     "numeric(%d, %d)",
	"time.Time-precision": "datetime(%d)",
}

// mysql dbBaser implementation.
type dbBaseMysql struct {
	dbBase
}

var _ dbBaser = new(dbBaseMysql)

// OperatorSQL Get mysql operator.
func (d *dbBaseMysql) OperatorSQL(operator string) string {
	return mysqlOperators[operator]
}

// DbTypes Get mysql table field types.
func (d *dbBaseMysql) DbTypes() map[string]string {
	return mysqlTypes
}

// ShowTablesQuery show table sql for mysql.
func (d *dbBaseMysql) ShowTablesQuery() string {
	return "SELECT table_name FROM information_schema.tables WHERE table_type = 'BASE TABLE' AND table_schema = DATABASE()"
}

// ShowColumnsQuery show Columns sql of table for mysql.
func (d *dbBaseMysql) ShowColumnsQuery(table string) string {
	return fmt.Sprintf("SELECT COLUMN_NAME, COLUMN_TYPE, IS_NULLABLE FROM information_schema.Columns "+
		"WHERE table_schema = DATABASE() AND table_name = '%s'", table)
}

// IndexExists execute sql to check index exist.
func (d *dbBaseMysql) IndexExists(ctx context.Context, db dbQuerier, table string, name string) bool {
	query := "SELECT count(*) FROM information_schema.statistics " +
		"WHERE table_schema = DATABASE() AND table_name = ? AND index_name = ?"
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

	row := db.QueryRowContext(ctx, fullQuery, table, name)
	var cnt int
	row.Scan(&cnt)
	return cnt > 0
}

// InsertOrUpdate a row
// If your primary key or unique column conflict will update
// If no will insert
// Add "`" for mysql sql building
func (d *dbBaseMysql) InsertOrUpdate(ctx context.Context, q dbQuerier, mi *models.ModelInfo, ind reflect.Value, a *alias, args ...string) (int64, error) {
	var iouStr string
	argsMap := map[string]string{}

	iouStr = "ON DUPLICATE KEY UPDATE"

	// Get on the key-value pairs
	for _, v := range args {
		kv := strings.Split(v, "=")
		if len(kv) == 2 {
			argsMap[strings.ToLower(kv[0])] = kv[1]
		}
	}

	names := make([]string, 0, len(mi.Fields.DBcols)-1)
	Q := d.ins.TableQuote()
	values, _, err := d.collectValues(mi, ind, mi.Fields.DBcols, true, true, &names, a.TZ)
	if err != nil {
		return 0, err
	}

	marks := make([]string, len(names))
	updateValues := make([]interface{}, 0)
	updates := make([]string, len(names))

	for i, v := range names {
		marks[i] = "?"
		valueStr := argsMap[strings.ToLower(v)]
		if valueStr != "" {
			updates[i] = "`" + v + "`" + "=" + valueStr
		} else {
			updates[i] = "`" + v + "`" + "=?"
			updateValues = append(updateValues, values[i])
		}
	}

	values = append(values, updateValues...)

	sep := fmt.Sprintf("%s, %s", Q, Q)
	qmarks := strings.Join(marks, ", ")
	qupdates := strings.Join(updates, ", ")
	columns := strings.Join(names, sep)

	// conflitValue maybe is an int,can`t use fmt.Sprintf
	query := fmt.Sprintf("INSERT INTO %s%s%s (%s%s%s) VALUES (%s) %s "+qupdates, Q, mi.Table, Q, Q, columns, Q, qmarks, iouStr)

	d.ins.ReplaceMarks(&query)

	// Prepend comments
	commentStr := q.GetQueryComments().String()
	fullQuery := commentStr + query

	// Note: HasReturningID might need adjustment if it relies on exact query prefix
	if !d.ins.HasReturningID(mi, &query) {
		res, err := q.ExecContext(ctx, fullQuery, values...) // Use fullQuery
		if err == nil {
			lastInsertId, err := res.LastInsertId()
			if err != nil {
				DebugLog.Println(ErrLastInsertIdUnavailable, ':', err)
				return lastInsertId, ErrLastInsertIdUnavailable
			} else {
				return lastInsertId, nil
			}
		}
		return 0, err
	}

	row := q.QueryRowContext(ctx, fullQuery, values...) // Use fullQuery
	var id int64
	err = row.Scan(&id)
	return id, err
}

// Helper method to handle comment prepending with type assertion
func (d *dbBaseMysql) prependCommentsIfSupported(db dbQuerier, query string) string {
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

// create new mysql dbBaser.
func newdbBaseMysql() dbBaser {
	b := new(dbBaseMysql)
	b.ins = b
	return b
}
