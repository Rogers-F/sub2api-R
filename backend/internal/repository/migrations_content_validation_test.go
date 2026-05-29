//go:build unit

package repository

import (
	"io/fs"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/migrations"
	"github.com/stretchr/testify/require"
)

// TestEmbeddedMigrations_PassRunnerParsing guards against migration files whose
// COMMENTS trip the runner's naive substring/`;`-based parsing — a class of bug
// that does not surface in unit tests but breaks startup migration application:
//   - a non-_notx file whose comment contains "CONCURRENTLY" is rejected;
//   - a _notx file whose comment contains BEGIN/COMMIT/ROLLBACK is rejected;
//   - a `;` inside a comment splits a statement mid-comment, leaking a non-SQL
//     fragment into the next executed statement.
//
// It runs the runner's real helpers against every embedded migration.
func TestEmbeddedMigrations_PassRunnerParsing(t *testing.T) {
	files, err := fs.Glob(migrations.FS, "*.sql")
	require.NoError(t, err)
	require.NotEmpty(t, files)

	for _, name := range files {
		content, err := fs.ReadFile(migrations.FS, name)
		require.NoError(t, err)

		// Execution-mode validation must accept every shipped migration.
		_, verr := validateMigrationExecutionMode(name, string(content))
		require.NoErrorf(t, verr, "migration %s failed execution-mode validation", name)

		// For non-transactional migrations, every executable (comment-stripped)
		// statement must be a clean CONCURRENTLY index op. A leaked comment
		// fragment (from a `;` inside a comment) would not begin with CREATE/DROP
		// INDEX and is caught here.
		if strings.HasSuffix(strings.ToLower(strings.TrimSpace(name)), nonTransactionalMigrationSuffix) {
			for _, stmt := range splitSQLStatements(string(content)) {
				s := stripSQLLineComment(strings.TrimSpace(stmt))
				if s == "" {
					continue
				}
				fields := strings.Fields(strings.ToUpper(s))
				require.GreaterOrEqualf(t, len(fields), 2, "malformed statement in %s: %q", name, s)
				require.Containsf(t, []string{"CREATE", "DROP"}, fields[0],
					"non-index statement leaked in %s: %q", name, s)
				require.Equalf(t, "INDEX", fields[1], "non-index statement leaked in %s: %q", name, s)
				require.Containsf(t, strings.ToUpper(s), "CONCURRENTLY",
					"_notx statement must be CONCURRENTLY in %s: %q", name, s)
			}
		}
	}
}
