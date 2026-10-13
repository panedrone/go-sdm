package pg

import (
	"errors"
	"net/url"
	"strings"
)

const jdbcPrefix = "jdbc:postgresql:"

// ConnString builds the pgx connection string from <jdbc url user pwd> of settings.xml.
// Both a JDBC URL (as in the Java plug-in) and a plain postgres:// are accepted.
func ConnString(jdbcURL, user, pwd string) (string, error) {
	raw := strings.TrimSpace(jdbcURL)
	switch {
	case strings.HasPrefix(raw, jdbcPrefix):
		raw = "postgres:" + raw[len(jdbcPrefix):]
	case strings.HasPrefix(raw, "postgres://"), strings.HasPrefix(raw, "postgresql://"):
	default:
		return "", errors.New("PostgreSQL URL expected (jdbc:postgresql://... or postgres://...): " + jdbcURL)
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	if u.Host == "" {
		u.Host = "localhost" // jdbc:postgresql:db
	}
	if u.User == nil && user != "" {
		if pwd != "" {
			u.User = url.UserPassword(user, pwd)
		} else {
			u.User = url.User(user)
		}
	}
	q := u.Query()
	// pgjdbc 'currentSchema' -> search_path, otherwise pgx would send it to the server as an unknown parameter
	if s := q.Get("currentSchema"); s != "" {
		q.Del("currentSchema")
		q.Set("search_path", s)
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}
