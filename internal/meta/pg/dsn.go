package pg

import (
	"errors"
	"net/url"
	"strings"
)

const jdbcPrefix = "jdbc:postgresql:"

// ConnString строит строку подключения pgx из <jdbc url user pwd> файла settings.xml.
// Принимается и JDBC-URL (как у Java-плагина), и обычный postgres://.
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
	// pgjdbc 'currentSchema' -> search_path, иначе pgx отправит его серверу как неизвестный параметр
	if s := q.Get("currentSchema"); s != "" {
		q.Del("currentSchema")
		q.Set("search_path", s)
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}
