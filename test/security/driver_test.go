package security_test

// The suite talks to a real database through database/sql, so it needs the
// driver registered. It is imported here rather than in the test file to keep
// the blank import obvious.
//
// This is a _test.go file on purpose. As a plain .go file it declared the
// directory's package name, and whether that clashed with the test files
// depended on ALPHABETICAL ORDER — adding a test file sorting before it broke
// the build with "found packages security and security_test".
import _ "github.com/jackc/pgx/v5/stdlib"
