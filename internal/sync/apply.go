package sync

import (
	"context"
	"fmt"

	"github.com/dalibo/ldap2pg/internal"
	"github.com/dalibo/ldap2pg/internal/perf"
	"github.com/dalibo/ldap2pg/internal/postgres"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/exp/slog"
)

func Apply(ctx context.Context, watch *perf.StopWatch, diff <-chan postgres.SyncQuery, real bool) (count int, err error) {
	formatter := postgres.FmtQueryRewriter{}

	prefix := ""
	if !real {
		prefix = "Would "
	}

	for query := range diff {
		slog.Log(ctx, internal.LevelChange, prefix+query.Description, query.LogArgs...)
		count++
		pgConn, err := postgres.DBPool.Get(ctx, query.Database)
		if err != nil {
			return count, fmt.Errorf("PostgreSQL error: %w", err)
		}

		// Rewrite query to log a pasteable query even when in Dry mode.
		sql, _, _ := formatter.RewriteQuery(ctx, pgConn, query.Query, query.QueryArgs)
		slog.Debug(prefix + "Execute SQL query:\n" + sql)

		if !real {
			continue
		}

		var tag pgconn.CommandTag
		duration := watch.TimeIt(func() {
			_, err = pgConn.Exec(ctx, sql)
		})
		if err != nil {
			return count, fmt.Errorf("sync: %w", err)
		}
		slog.Debug("Query terminated.", "duration", duration, "rows", tag.RowsAffected())
	}
	return
}
