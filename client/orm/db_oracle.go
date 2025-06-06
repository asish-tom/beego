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
	"strings"

	"github.com/asish-tom/beego/v2/client/orm/internal/models"

	"github.com/asish-tom/beego/v2/client/orm/hints"
)

// oracle operators.
var oracleOperators = map[string]string{
	"exact":       "= ?",
	"gt":          "> ?",
	"gte":         ">= ?",
	"lt":          "< ?",
	"lte":         "<= ?",
	"//iendswith": "LIKE ?",
}

// oracle column field types.
var oracleTypes = map[string]string{
	"pk":                  "NOT NULL PRIMARY KEY",
	"bool":                "bool",
	"string":              "VARCHAR2(%d)",
	"string-char":         "CHAR(%d)",
	"string-text":         "VARCHAR2(%d)",
	"time.Time-date":      "DATE",
	"time.Time":           "TIMESTAMP",
	"int8":                "INTEGER",
	"int16":               "INTEGER",
	"int32":               "INTEGER",
	"int64":               "INTEGER",
	"uint8":               "INTEGER",
	"uint16":              "INTEGER",
	"uint32":              "INTEGER",
	"uint64":              "INTEGER",
	"float64":             "NUMBER",
	"float64-decimal":     "NUMBER(%d, %d)",
	"time.Time-precision": "TIMESTAMP(%d)",
}

// oracle dbBaser
type dbBaseOracle struct {
	dbBase
}

var _ dbBaser = new(dbBaseOracle)

// Helper method to handle comment prepending with type assertion
func (d *dbBaseOracle) prependCommentsIfSupported(db dbQuerier, query string) string {
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

// create oracle dbBaser.
func newdbBaseOracle() dbBaser {
	b := new(dbBaseOracle)
	b.ins = b
	return b
}

// OperatorSQL Get oracle operator.
func (d *dbBaseOracle) OperatorSQL(operator string) string {
	return oracleOperators[operator]
}

// DbTypes Get oracle table field types.
func (d *dbBaseOracle) DbTypes() map[string]string {
	return oracleTypes
}

// ShowTablesQuery show All the tables in database
func (d *dbBaseOracle) ShowTablesQuery() string {
	return "SELECT TABLE_NAME FROM USER_TABLES"
}

// Oracle
func (d *dbBaseOracle) ShowColumnsQuery(table string) string {
	return fmt.Sprintf("SELECT COLUMN_NAME, DATA_TYPE, NULLABLE FROM ALL_TAB_COLUMNS "+
		"WHERE TABLE_NAME ='%s'", strings.ToUpper(table))
}

// check index is exist
func (d *dbBaseOracle) IndexExists(ctx context.Context, db dbQuerier, table string, name string) bool {
	query := "SELECT COUNT(*) FROM USER_IND_COLUMNS, USER_INDEXES " +
		"WHERE USER_IND_COLUMNS.INDEX_NAME = USER_INDEXES.INDEX_NAME " +
		"AND  USER_IND_COLUMNS.TABLE_NAME = ? AND USER_IND_COLUMNS.INDEX_NAME = ?"

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

	row := db.QueryRowContext(ctx, fullQuery, strings.ToUpper(table), strings.ToUpper(name))

	var cnt int
	row.Scan(&cnt)
	return cnt > 0
}

func (d *dbBaseOracle) GenerateSpecifyIndex(tableName string, useIndex int, indexes []string) string {
	var s []string
	Q := d.TableQuote()
	for _, index := range indexes {
		tmp := fmt.Sprintf(`%s%s%s`, Q, index, Q)
		s = append(s, tmp)
	}

	var hint string

	switch useIndex {
	case hints.KeyUseIndex, hints.KeyForceIndex:
		hint = `INDEX`
	case hints.KeyIgnoreIndex:
		hint = `NO_INDEX`
	default:
		DebugLog.Println("[WARN] Not a valid specifying action, so that action is ignored")
		return ``
	}

	return fmt.Sprintf(` /*+ %s(%s %s)*/ `, hint, tableName, strings.Join(s, `,`))
}

// InsertValue execute insert sql with given struct and given values.
// insert the given values, not the field values in struct.
func (d *dbBaseOracle) InsertValue(ctx context.Context, q dbQuerier, mi *models.ModelInfo, isMulti bool, names []string, values []interface{}) (int64, error) {
	Q := d.ins.TableQuote()

	marks := make([]string, len(names))
	for i := range marks {
		marks[i] = ":" + names[i]
	}

	sep := fmt.Sprintf("%s, %s", Q, Q)
	qmarks := strings.Join(marks, ", ")
	columns := strings.Join(names, sep)

	multi := len(values) / len(names)

	if isMulti {
		qmarks = strings.Repeat(qmarks+"), (", multi-1) + qmarks
	}

	query := fmt.Sprintf("INSERT INTO %s%s%s (%s%s%s) VALUES (%s)", Q, mi.Table, Q, Q, columns, Q, qmarks)

	d.ins.ReplaceMarks(&query)

	// Prepend comments
	commentStr := q.GetQueryComments().String()
	fullQuery := commentStr + query

	// Note: HasReturningID might need adjustment if it relies on exact query prefix
	if isMulti || !d.ins.HasReturningID(mi, &query) { // Check original query for HasReturningID
		res, err := q.ExecContext(ctx, fullQuery, values...) // Use fullQuery
		if err == nil {
			if isMulti {
				return res.RowsAffected()
			}

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
	err := row.Scan(&id)
	return id, err
}
