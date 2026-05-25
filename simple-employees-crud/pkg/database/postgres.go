// Package database provides the PostgreSQL connection pool and helpers
// for transaction management.  The DBTX interface abstracts over both
// *pgxpool.Pool and pgx.Tx so repository methods work identically whether
// called inside or outside a transaction.
package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"simple-employees-crud/pkg/apperror"
	"simple-employees-crud/pkg/logger"
)

// DBTX is satisfied by both *pgxpool.Pool and pgx.Tx, allowing repository
// methods to be called inside or outside a transaction without duplication.
type DBTX interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// txKey is an unexported context key for the active pgx.Tx.
type txKey struct{}

// PoolConfig holds parameters used to configure the connection pool.
type PoolConfig struct {
	DSN             string
	MaxConns        int32
	MinConns        int32
	MaxConnLifetime time.Duration
}

// NewPool opens and validates a pgxpool connection pool.
func NewPool(ctx context.Context, cfg PoolConfig) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("database: invalid DSN: %w", err)
	}

	if cfg.MaxConns > 0 {
		poolCfg.MaxConns = cfg.MaxConns
	}
	if cfg.MinConns > 0 {
		poolCfg.MinConns = cfg.MinConns
	}
	if cfg.MaxConnLifetime > 0 {
		poolCfg.MaxConnLifetime = cfg.MaxConnLifetime
	}

	// Trace slow queries to the application logger.
	poolCfg.ConnConfig.Tracer = &queryTracer{}

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("database: cannot create pool: %w", err)
	}

	if err = pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("database: ping failed: %w", err)
	}

	logger.S().Infow("PostgreSQL pool connected",
		"max_conns", poolCfg.MaxConns,
		"min_conns", poolCfg.MinConns,
	)
	return pool, nil
}

// ─── Transaction helpers ─────────────────────────────────────────────────────

// WithTx begins a transaction, passes a context carrying the tx to fn, and
// commits or rolls back based on whether fn returns an error.
// Repositories call FromContext to obtain the active DBTX transparently.
func WithTx(ctx context.Context, pool *pgxpool.Pool, fn func(context.Context) error) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return apperror.NewInternal("failed to begin transaction")
	}

	// Rollback on panic so deferred tx.Rollback is always called.
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p)
		}
	}()

	txCtx := context.WithValue(ctx, txKey{}, tx)
	if err := fn(txCtx); err != nil {
		_ = tx.Rollback(ctx)
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return apperror.NewInternal("failed to commit transaction")
	}
	return nil
}

// FromContext extracts the active pgx.Tx from ctx (if present) and returns it
// as a DBTX; otherwise it returns the pool.  Repositories call this in every
// method so they participate in transactions automatically.
func FromContext(ctx context.Context, pool *pgxpool.Pool) DBTX {
	if tx, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return tx
	}
	return pool
}

// ─── Query tracer (slow-query logging) ───────────────────────────────────────

type queryTracer struct{}

func (t *queryTracer) TraceQueryStart(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	ctx = context.WithValue(ctx, struct{ key string }{"sql_start"}, time.Now())
	return context.WithValue(ctx, struct{ key string }{"sql_query"}, data.SQL)
}

func (t *queryTracer) TraceQueryEnd(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryEndData) {
	start, ok := ctx.Value(struct{ key string }{"sql_start"}).(time.Time)
	if !ok {
		return
	}
	sql, _ := ctx.Value(struct{ key string }{"sql_query"}).(string)
	elapsed := time.Since(start)
	if elapsed > 200*time.Millisecond {
		logger.S().Warnw("slow query detected",
			"sql", sql,
			"duration_ms", elapsed.Milliseconds(),
		)
	}
}
