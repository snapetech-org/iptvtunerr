package plex

import "strings"

// quoteSQLiteIdentifier safely quotes a SQLite table or column name. SQLite
// cannot bind identifiers as query parameters; names discovered from Plex's
// database schema must therefore be escaped before they are interpolated.
func quoteSQLiteIdentifier(identifier string) string {
	return `"` + strings.ReplaceAll(identifier, `"`, `""`) + `"`
}
