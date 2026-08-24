// Package db 负责连接池、迁移与种子。
// 双副本同时 AutoMigrate 可能在 pg_type 上撞 23505，因此用 pg_advisory_lock(20260823) 串行化。
package db

import (
	"fmt"
	"time"

	"gocooking/internal/model"
	"gocooking/pkg/logger"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	glogger "gorm.io/gorm/logger"
)

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

func acquireMigrationLock(gdb *gorm.DB) (func(), error) {
	sqlDB, err := gdb.DB()
	if err != nil {
		return nil, err
	}
	if _, err := sqlDB.Exec("SELECT pg_advisory_lock(20260823)"); err != nil {
		return nil, fmt.Errorf("advisory lock: %w", err)
	}
	release := func() {
		_, _ = sqlDB.Exec("SELECT pg_advisory_unlock(20260823)")
	}
	defer release()
	return release, nil
}
