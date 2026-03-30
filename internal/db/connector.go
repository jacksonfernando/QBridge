package db

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/jacksonfernando/qbridge/internal/config"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

// Connect opens a *sql.DB connection for the given Database config.
func Connect(d *config.Database) (*sql.DB, error) {
	switch d.Type {
	case config.DBTypePostgres:
		return connectPostgres(d)
	case config.DBTypeMySQL:
		return connectMySQL(d)
	case config.DBTypeSQLite:
		return connectSQLite(d)
	default:
		return nil, fmt.Errorf("unsupported database type: %q", d.Type)
	}
}

func connectPostgres(d *config.Database) (*sql.DB, error) {
	sslmode := d.SSLMode
	if sslmode == "" {
		sslmode = "prefer"
	}
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.DBName, sslmode,
	)
	conn, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("postgres: %w", err)
	}
	if err := conn.Ping(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("postgres connection failed: %w", err)
	}
	return conn, nil
}

func connectMySQL(d *config.Database) (*sql.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true",
		d.User, d.Password, d.Host, d.Port, d.DBName,
	)
	conn, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("mysql: %w", err)
	}
	if err := conn.Ping(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("mysql connection failed: %w", err)
	}
	return conn, nil
}

func connectSQLite(d *config.Database) (*sql.DB, error) {
	if d.FilePath == "" {
		return nil, fmt.Errorf("sqlite: filepath is required")
	}
	conn, err := sql.Open("sqlite3", d.FilePath)
	if err != nil {
		return nil, fmt.Errorf("sqlite: %w", err)
	}
	if err := conn.Ping(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("sqlite connection failed: %w", err)
	}
	return conn, nil
}

// QueryResult holds the result of a SQL query.
type QueryResult struct {
	Columns []string         `json:"columns"`
	Rows    [][]interface{}  `json:"rows"`
	Affected int64           `json:"rows_affected,omitempty"`
}

// Execute runs a SQL statement and returns structured results.
func Execute(conn *sql.DB, query string) (*QueryResult, error) {
	query = strings.TrimSpace(query)

	// Determine if this is a SELECT-like statement that returns rows.
	keyword := strings.ToUpper(strings.Fields(query)[0])
	if keyword == "SELECT" || keyword == "WITH" {
		return queryRows(conn, query)
	}
	return execStatement(conn, query)
}

func queryRows(conn *sql.DB, query string) (*QueryResult, error) {
	rows, err := conn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	result := &QueryResult{Columns: cols, Rows: [][]interface{}{}}

	for rows.Next() {
		vals := make([]interface{}, len(cols))
		ptrs := make([]interface{}, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		row := make([]interface{}, len(cols))
		for i, v := range vals {
			// Convert []byte to string for readability.
			if b, ok := v.([]byte); ok {
				row[i] = string(b)
			} else {
				row[i] = v
			}
		}
		result.Rows = append(result.Rows, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func execStatement(conn *sql.DB, query string) (*QueryResult, error) {
	res, err := conn.Exec(query)
	if err != nil {
		return nil, err
	}
	affected, _ := res.RowsAffected()
	return &QueryResult{Columns: []string{}, Rows: [][]interface{}{}, Affected: affected}, nil
}
