package cache

import (
	"context"
	"hash/fnv"
	"os"
	"strconv"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/callbacks"
)

type Config struct {
	Store Store

	Prefix string

	Serializer Serializer
}

type (
	Serializer interface {
		Serialize(v any) ([]byte, error)

		Deserialize(data []byte, v any) error
	}

	Store interface {
		Set(ctx context.Context, key string, value any, ttl time.Duration) error

		Get(ctx context.Context, key string) ([]byte, error)
	}
)

type Cache struct {
	store Store

	Serializer Serializer

	prefix string
}

func New(conf *Config) *Cache {
	if conf.Store == nil {
		os.Exit(1)
	}

	if conf.Serializer == nil {
		conf.Serializer = &DefaultJSONSerializer{}
	}

	return &Cache{
		store:      conf.Store,
		prefix:     conf.Prefix,
		Serializer: conf.Serializer,
	}
}

func (p *Cache) Name() string {
	return "gorm:cache"
}

func (p *Cache) Initialize(tx *gorm.DB) error {
	return tx.Callback().Query().Replace("gorm:query", p.Query)
}

func generateKey(key string) string {
	hash := fnv.New64a()
	_, _ = hash.Write([]byte(key))

	return strconv.FormatUint(hash.Sum64(), 36)
}

func (p *Cache) Query(tx *gorm.DB) {
	ctx := tx.Statement.Context

	ttl, ok := FromExpiration(ctx)

	if !ok {
		callbacks.Query(tx)
		return
	}

	var key string
	var hasKey bool

	// 调用 Gorm的方法生产SQL
	callbacks.BuildQuerySQL(tx)

	// 是否有自定义key
	if key, hasKey = FromKey(ctx); !hasKey {
		key = p.prefix + generateKey(tx.Statement.SQL.String())
	}

	// 查询缓存数据
	if err := p.QueryCache(ctx, key, &tx.Statement.Dest); err == nil {
		return
	}

	// 查询数据库
	p.QueryDB(tx)
	if tx.Error != nil {
		return
	}

	// 写入缓存
	if err := p.SaveCache(ctx, key, tx.Statement.Dest, ttl); err != nil {
		tx.Logger.Error(ctx, err.Error())
	}
}

// QueryDB
// 这里重写Query方法 是不想执行 callbacks.BuildQuerySQL 两遍
func (p *Cache) QueryDB(tx *gorm.DB) {
	if tx.Error != nil || tx.DryRun {
		return
	}

	rows, err := tx.Statement.ConnPool.QueryContext(tx.Statement.Context, tx.Statement.SQL.String(), tx.Statement.Vars...)
	if err != nil {
		_ = tx.AddError(err)
		return
	}

	defer func() {
		_ = tx.AddError(rows.Close())
	}()

	gorm.Scan(rows, tx, 0)
}

func (p *Cache) QueryCache(ctx context.Context, key string, dest any) error {
	values, err := p.store.Get(ctx, key)
	if err != nil {
		return err
	}

	return p.Serializer.Deserialize(values, dest)
}

// SaveCache
// 写入缓存数据
func (p *Cache) SaveCache(ctx context.Context, key string, dest any, ttl time.Duration) error {
	values, err := p.Serializer.Serialize(dest)
	if err != nil {
		return err
	}

	return p.store.Set(ctx, key, values, ttl)
}
