package security_test

// The suite talks to a real database through database/sql, so it needs the
// driver registered. It is imported here rather than in the test file to keep
// the blank import obvious.
import _ "github.com/jackc/pgx/v5/stdlib"
