// Package db 负责连接池、迁移与种子。
// 双副本同时 AutoMigrate 可能在 pg_type 上撞 23505，因此用 pg_advisory_lock 串行化。
package db

import (
	"context"
	"fmt"
	"time"

	"gocooking/internal/model"
	"gocooking/pkg/logger"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	glogger "gorm.io/gorm/logger"
)

// migrationLockKey 是 advisory lock 的全局 key，所有实例共用。
// 任意两个实例同时迁移时，后到者会阻塞在前到者释放之前。
const migrationLockKey int64 = 20260823

func Open(dsn string) (*gorm.DB, error) {
	// 连接池刻意偏小：单用户工具，16 已足够，避免打满 Postgres。
	// MaxLifetime 30m，避免对端 Recycle 后把死连接交给请求。

	gdb, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: glogger.Default.LogMode(glogger.Warn),
	})
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}
	sqlDB, err := gdb.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(16)
	sqlDB.SetMaxIdleConns(4)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)
	return gdb, nil
}

func Migrate(gdb *gorm.DB) error {
	unlock, err := acquireMigrationLock(gdb)
	if err != nil {
		return err
	}
	defer unlock()

	if err := gdb.AutoMigrate(
		&model.User{},
		&model.Ingredient{},
		&model.Recipe{},
		&model.RecipeItem{},
		&model.MealSlot{},
		&model.PantryItem{},
		&model.StapleOverride{},
		&model.ShoppingCheck{},
	); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	logger.Info("db migrated")
	return nil
}

// acquireMigrationLock 使用 pg_advisory_lock 串行化多实例迁移。
//
// 关键实现细节：
//   - 用 sql.DB.Conn(ctx) 钉住一条连接（= 一个 Postgres session），
//     保证 lock 和 unlock 在同一 session 上执行。
//     如果直接用 sqlDB.Exec，连接池可能把 unlock 路由到另一条 session，
//     该 session 未持锁 → unlock 报错或静默失败，锁泄漏。
//   - lock 在 acquire 时获取、在返回的 release 闭包里释放，
//     中间不 defer release()，确保锁在 AutoMigrate 全程持有。
//   - release 返回前不 Close 连接，由调用方 defer unlock() 保证。
func acquireMigrationLock(gdb *gorm.DB) (func(), error) {
	sqlDB, err := gdb.DB()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	conn, err := sqlDB.Conn(ctx)
	if err != nil {
		return nil, fmt.Errorf("acquire connection for advisory lock: %w", err)
	}

	if _, err := conn.ExecContext(ctx, "SELECT pg_advisory_lock($1)", migrationLockKey); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("advisory lock: %w", err)
	}

	released := false
	return func() {
		if released {
			return
		}
		released = true
		unlockCtx, unlockCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer unlockCancel()
		_, _ = conn.ExecContext(unlockCtx, "SELECT pg_advisory_unlock($1)", migrationLockKey)
		_ = conn.Close()
	}, nil
}
