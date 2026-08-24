package db_test

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	appdb "gocooking/internal/db"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestMigrateSerializesConcurrentSchemaChanges(t *testing.T) {
	locks := newAdvisoryLocks()
	sqlDB := sql.OpenDB(advisoryConnector{locks: locks})
	t.Cleanup(func() { _ = sqlDB.Close() })

	gate := newMigrationGate()
	dialector := migrationDialector{
		Dialector: postgres.New(postgres.Config{WithoutReturning: true}),
		pool:      sqlDB,
		gate:      gate,
	}
	gdb, err := gorm.Open(dialector, &gorm.Config{DisableAutomaticPing: true})
	if err != nil {
		t.Fatalf("open controlled database: %v", err)
	}

	errs := make(chan error, 2)
	go func() { errs <- appdb.Migrate(gdb) }()
	select {
	case <-gate.firstStarted:
	case <-time.After(time.Second):
		t.Fatal("first migration did not start")
	}

	go func() { errs <- appdb.Migrate(gdb) }()
	overlapped := false
	select {
	case <-gate.secondStarted:
		overlapped = true
	case <-time.After(100 * time.Millisecond):
	}
	close(gate.releaseFirst)

	for i := 0; i < 2; i++ {
		select {
		case err := <-errs:
			if err != nil {
				t.Fatalf("migration returned error: %v", err)
			}
		case <-time.After(time.Second):
			t.Fatal("migration did not finish")
		}
	}
	if overlapped {
		t.Fatal("second schema migration started before the first migration finished")
	}
}

type migrationGate struct {
	mu            sync.Mutex
	calls         int
	firstStarted  chan struct{}
	releaseFirst  chan struct{}
	secondStarted chan struct{}
}

func newMigrationGate() *migrationGate {
	return &migrationGate{
		firstStarted:  make(chan struct{}),
		releaseFirst:  make(chan struct{}),
		secondStarted: make(chan struct{}),
	}
}

type migrationMigrator struct {
	gorm.Migrator
	gate *migrationGate
}

func (m migrationMigrator) AutoMigrate(...any) error {
	m.gate.mu.Lock()
	m.gate.calls++
	call := m.gate.calls
	m.gate.mu.Unlock()

	switch call {
	case 1:
		close(m.gate.firstStarted)
		<-m.gate.releaseFirst
	case 2:
		close(m.gate.secondStarted)
	}
	return nil
}

type migrationDialector struct {
	gorm.Dialector
	pool *sql.DB
	gate *migrationGate
}

func (d migrationDialector) Initialize(db *gorm.DB) error {
	db.ConnPool = d.pool
	return nil
}

func (d migrationDialector) Migrator(*gorm.DB) gorm.Migrator {
	return migrationMigrator{gate: d.gate}
}

type advisoryLocks struct {
	token chan struct{}
}

func newAdvisoryLocks() *advisoryLocks {
	locks := &advisoryLocks{token: make(chan struct{}, 1)}
	locks.token <- struct{}{}
	return locks
}

type advisoryConnector struct {
	locks *advisoryLocks
}

func (c advisoryConnector) Connect(context.Context) (driver.Conn, error) {
	return &advisoryConn{locks: c.locks}, nil
}

func (c advisoryConnector) Driver() driver.Driver {
	return advisoryDriver{locks: c.locks}
}

type advisoryDriver struct {
	locks *advisoryLocks
}

func (d advisoryDriver) Open(string) (driver.Conn, error) {
	return &advisoryConn{locks: d.locks}, nil
}

type advisoryConn struct {
	locks *advisoryLocks
}

func (c *advisoryConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("prepare is not supported")
}

func (c *advisoryConn) Close() error { return nil }

func (c *advisoryConn) Begin() (driver.Tx, error) {
	return nil, errors.New("transactions are not supported")
}

func (c *advisoryConn) ExecContext(ctx context.Context, query string, _ []driver.NamedValue) (driver.Result, error) {
	switch {
	case strings.Contains(query, "pg_advisory_unlock"):
		select {
		case c.locks.token <- struct{}{}:
			return driver.RowsAffected(1), nil
		default:
			return nil, errors.New("advisory lock is not held")
		}
	case strings.Contains(query, "pg_advisory_lock"):
		select {
		case <-c.locks.token:
			return driver.RowsAffected(1), nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	default:
		return nil, errors.New("unexpected query")
	}
}
