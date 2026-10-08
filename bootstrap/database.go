package bootstrap

import (
	"database/sql"
	"errors"
	"log/slog"
	"miltechserver/helper"
	"net"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

func NewSqlClient(env *Env) *sql.DB {
	dsnStr, err := DatabaseDSN(env)
	helper.PanicOnError(err)
	slog.Info("Connecting to Database")
	db, err := sql.Open("postgres", dsnStr)
	if err != nil {
		panic("invalid database connection configuration")
	}

	err = db.Ping()

	if err != nil {
		_ = db.Close()
		slog.Error("Unable to connect to database")
		panic("unable to connect to database")
	}

	slog.Info("Connected to Database!")

	// Configure connection pool for parallel query workloads
	db.SetMaxOpenConns(env.DBMaxOpenConns)
	db.SetMaxIdleConns(env.DBMaxIdleConns)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(1 * time.Minute)

	slog.Info("Connection pool configured",
		"maxOpenConns", env.DBMaxOpenConns,
		"maxIdleConns", env.DBMaxIdleConns,
		"connMaxLifetime", "5m",
		"connMaxIdleTime", "1m")

	return db
}

var databaseSchemaName = regexp.MustCompile(`^[a-z_][a-z0-9_]*$`)

// DatabaseDSN is shared by all application connections; URL escaping preserves credentials literally.
func DatabaseDSN(env *Env) (string, error) {
	if env == nil {
		return "", errors.New("invalid database configuration")
	}
	port, err := strconv.Atoi(env.Port)
	if err != nil || port < 1 || port > 65535 || env.Host == "" || env.Username == "" || env.DBName == "" || !databaseSchemaName.MatchString(env.DBSchema) || strings.ContainsAny(env.Host, "/ ?#") {
		return "", errors.New("invalid database configuration")
	}
	switch env.SslMode {
	case "disable", "require", "verify-ca", "verify-full":
	default:
		return "", errors.New("invalid database TLS configuration")
	}
	u := url.URL{Scheme: "postgresql", Host: net.JoinHostPort(env.Host, env.Port), Path: "/" + env.DBName, User: url.UserPassword(env.Username, env.Password)}
	query := url.Values{"sslmode": {env.SslMode}, "search_path": {env.DBSchema}}
	u.RawQuery = query.Encode()
	return u.String(), nil
}
