package app

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Row adalah satu baris hasil query, mirip array asosiatif PHP.
type Row map[string]any

func (r Row) Has(k string) bool { _, ok := r[k]; return ok }

func (r Row) S(k string) string {
	if r == nil {
		return ""
	}
	return toStr(r[k])
}

func (r Row) I(k string) int64 {
	if r == nil {
		return 0
	}
	return toInt(r[k])
}

func (r Row) F(k string) float64 {
	if r == nil {
		return 0
	}
	return toFloat(r[k])
}

func (r Row) Null(k string) bool { return r == nil || r[k] == nil }

func toStr(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	case []byte:
		return string(x)
	case int64:
		return strconv.FormatInt(x, 10)
	case int:
		return strconv.Itoa(x)
	case float64:
		return phpFloatStr(x)
	case bool:
		if x {
			return "1"
		}
		return ""
	default:
		return fmt.Sprint(x)
	}
}

// phpFloatStr meniru (string) $float di PHP.
func phpFloatStr(f float64) string {
	if f == math.Trunc(f) && math.Abs(f) < 1e15 {
		return strconv.FormatInt(int64(f), 10)
	}
	return strconv.FormatFloat(f, 'G', 14, 64)
}

func toInt(v any) int64 {
	switch x := v.(type) {
	case nil:
		return 0
	case int64:
		return x
	case int:
		return int64(x)
	case float64:
		return int64(x)
	case bool:
		if x {
			return 1
		}
		return 0
	default:
		return int64(toFloat(x))
	}
}

func toFloat(v any) float64 {
	switch x := v.(type) {
	case nil:
		return 0
	case float64:
		return x
	case int64:
		return float64(x)
	case int:
		return float64(x)
	case bool:
		if x {
			return 1
		}
		return 0
	default:
		s := strings.TrimSpace(toStr(x))
		// PHP (float)"12abc" = 12: ambil awalan numerik.
		end := 0
		for i, c := range s {
			if (c >= '0' && c <= '9') || c == '.' || ((c == '-' || c == '+') && i == 0) || c == 'e' || c == 'E' {
				end = i + 1
				continue
			}
			break
		}
		for end > 0 {
			if f, err := strconv.ParseFloat(s[:end], 64); err == nil {
				return f
			}
			end--
		}
		return 0
	}
}

type querier interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

// DB membungkus koneksi SQLite. Satu koneksi saja agar transaksi dan PRAGMA konsisten.
type DB struct {
	sqlDB *sql.DB
	q     querier
	tx    *sql.Tx
}

var Q *DB

func openDB(path string) (*DB, error) {
	dsn := "file:" + path + "?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)"
	s, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	s.SetMaxOpenConns(1)
	s.SetMaxIdleConns(1)
	s.SetConnMaxLifetime(0)
	if err := s.Ping(); err != nil {
		return nil, err
	}
	return &DB{sqlDB: s, q: s}, nil
}

func (d *DB) Close() error { return d.sqlDB.Close() }

func normArgs(args []any) []any {
	out := make([]any, len(args))
	for i, a := range args {
		switch x := a.(type) {
		case int:
			out[i] = int64(x)
		case bool:
			if x {
				out[i] = int64(1)
			} else {
				out[i] = int64(0)
			}
		default:
			out[i] = a
		}
	}
	return out
}

func (d *DB) Run(q string, args ...any) sql.Result {
	res, err := d.q.ExecContext(context.Background(), q, normArgs(args)...)
	if err != nil {
		panic(fmt.Errorf("sql: %w\n%s", err, q))
	}
	return res
}

func (d *DB) All(q string, args ...any) []Row {
	rows, err := d.q.QueryContext(context.Background(), q, normArgs(args)...)
	if err != nil {
		panic(fmt.Errorf("sql: %w\n%s", err, q))
	}
	defer rows.Close()
	cols, _ := rows.Columns()
	out := []Row{}
	for rows.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			panic(err)
		}
		r := Row{}
		for i, c := range cols {
			if b, ok := vals[i].([]byte); ok {
				r[c] = string(b)
			} else {
				r[c] = vals[i]
			}
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		panic(err)
	}
	return out
}

func (d *DB) One(q string, args ...any) Row {
	rows := d.All(q, args...)
	if len(rows) == 0 {
		return nil
	}
	return rows[0]
}

// Value mengembalikan kolom pertama baris pertama, atau nil.
func (d *DB) Value(q string, args ...any) any {
	rows, err := d.q.QueryContext(context.Background(), q, normArgs(args)...)
	if err != nil {
		panic(fmt.Errorf("sql: %w\n%s", err, q))
	}
	defer rows.Close()
	if !rows.Next() {
		return nil
	}
	var v any
	cols, _ := rows.Columns()
	vals := make([]any, len(cols))
	ptrs := make([]any, len(cols))
	for i := range vals {
		ptrs[i] = &vals[i]
	}
	if err := rows.Scan(ptrs...); err != nil {
		panic(err)
	}
	v = vals[0]
	if b, ok := v.([]byte); ok {
		return string(b)
	}
	return v
}

func (d *DB) Int(q string, args ...any) int64     { return toInt(d.Value(q, args...)) }
func (d *DB) Float(q string, args ...any) float64 { return toFloat(d.Value(q, args...)) }
func (d *DB) Str(q string, args ...any) string    { return toStr(d.Value(q, args...)) }

func sortedKeys(m Row) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func (d *DB) Insert(table string, data Row) int64 {
	keys := sortedKeys(data)
	args := make([]any, len(keys))
	ph := make([]string, len(keys))
	for i, k := range keys {
		args[i] = data[k]
		ph[i] = "?"
	}
	res := d.Run(fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", table, strings.Join(keys, ", "), strings.Join(ph, ", ")), args...)
	id, _ := res.LastInsertId()
	return id
}

func (d *DB) Update(table string, data Row, id int64) {
	keys := sortedKeys(data)
	sets := make([]string, len(keys))
	args := make([]any, 0, len(keys)+1)
	for i, k := range keys {
		sets[i] = k + " = ?"
		args = append(args, data[k])
	}
	args = append(args, id)
	d.Run(fmt.Sprintf("UPDATE %s SET %s WHERE id = ?", table, strings.Join(sets, ", ")), args...)
}

// Tx menjalankan fn dalam transaksi; panic di dalam fn membatalkan transaksi lalu diteruskan.
func (d *DB) Tx(fn func()) {
	if d.tx != nil {
		fn()
		return
	}
	tx, err := d.sqlDB.Begin()
	if err != nil {
		panic(err)
	}
	d.tx, d.q = tx, tx
	done := false
	defer func() {
		d.tx, d.q = nil, d.sqlDB
		if !done {
			_ = tx.Rollback()
		}
	}()
	fn()
	if err := tx.Commit(); err != nil {
		panic(err)
	}
	done = true
}

func (d *DB) Migrate() error {
	if _, err := d.sqlDB.Exec(`CREATE TABLE IF NOT EXISTS migrations (name TEXT PRIMARY KEY, applied_at TEXT NOT NULL)`); err != nil {
		return err
	}
	applied := map[string]bool{}
	for _, r := range d.All(`SELECT name FROM migrations`) {
		applied[r.S("name")] = true
	}
	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return err
	}
	names := []string{}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".sql") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	for _, name := range names {
		if applied[name] {
			continue
		}
		body, _ := migrationsFS.ReadFile("migrations/" + name)
		tx, err := d.sqlDB.Begin()
		if err != nil {
			return err
		}
		if _, err := tx.Exec(string(body)); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("migrasi %s: %w", name, err)
		}
		if _, err := tx.Exec(`INSERT INTO migrations (name, applied_at) VALUES (?, ?)`, name, time.Now().Format("2006-01-02T15:04:05-07:00")); err != nil {
			_ = tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}
