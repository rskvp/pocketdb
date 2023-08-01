package gormx

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
	"gorm.io/plugin/dbresolver"
)

type ResolverConfig struct {
	DBType   string // mysql
	Sources  []string
	Replicas []string
	Tables   []string
}

type Config struct {
	Debug        bool
	DBType       string // sqlite3
	DSN          string
	MaxLifetime  int
	MaxIdleTime  int
	MaxOpenConns int
	MaxIdleConns int
	TablePrefix  string
	Resolver     []ResolverConfig
}

func New(c Config) (*gorm.DB, error) {
	var dialector gorm.Dialector

	switch strings.ToLower(c.DBType) {

	case "sqlite3":
		_ = os.MkdirAll(filepath.Dir(c.DSN), os.ModePerm)
		dialector = sqlite.Open(c.DSN)
	default:
		return nil, fmt.Errorf("unsupported database type: %s", c.DBType)
	}

	gconfig := &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			TablePrefix:   c.TablePrefix,
			SingularTable: true,
		},
		Logger: logger.Discard,
	}

	if c.Debug {
		gconfig.Logger = logger.Default
	}

	db, err := gorm.Open(dialector, gconfig)
	if err != nil {
		return nil, err
	}

	if len(c.Resolver) > 0 {
		resolver := &dbresolver.DBResolver{}
		for _, r := range c.Resolver {
			rcfg := dbresolver.Config{}

			var open func(dsn string) gorm.Dialector
			dbType := strings.ToLower(r.DBType)
			switch dbType {

			case "sqlite3":
				open = sqlite.Open
			default:
				continue
			}

			for _, replica := range r.Replicas {
				if dbType == "sqlite3" {
					_ = os.MkdirAll(filepath.Dir(c.DSN), os.ModePerm)
				}
				rcfg.Replicas = append(rcfg.Replicas, open(replica))
			}
			for _, source := range r.Sources {
				if dbType == "sqlite3" {
					_ = os.MkdirAll(filepath.Dir(c.DSN), os.ModePerm)
				}
				rcfg.Sources = append(rcfg.Sources, open(source))
			}
			tables := stringSliceToInterfaceSlice(r.Tables)
			resolver.Register(rcfg, tables...)
			zap.L().Info(fmt.Sprintf("Use resolver, #tables: %v, #replicas: %v, #sources: %v \n",
				tables, r.Replicas, r.Sources))
		}

		resolver.SetMaxIdleConns(c.MaxIdleConns).
			SetMaxOpenConns(c.MaxOpenConns).
			SetConnMaxLifetime(time.Duration(c.MaxLifetime) * time.Second).
			SetConnMaxIdleTime(time.Duration(c.MaxIdleTime) * time.Second)
		if err := db.Use(resolver); err != nil {
			return nil, err
		}
	}

	if c.Debug {
		db = db.Debug()
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	sqlDB.SetMaxIdleConns(c.MaxIdleConns)
	sqlDB.SetMaxOpenConns(c.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(time.Duration(c.MaxLifetime) * time.Second)
	sqlDB.SetConnMaxIdleTime(time.Duration(c.MaxIdleTime) * time.Second)

	return db, nil
}

func stringSliceToInterfaceSlice(s []string) []interface{} {
	r := make([]interface{}, len(s))
	for i, v := range s {
		r[i] = v
	}
	return r
}
