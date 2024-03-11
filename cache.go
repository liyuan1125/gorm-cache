package cache

import (
	"context"
	"hash/fnv"
	"os"
	"strconv"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/callbacks"
	"gorm.io/gorm/logger"
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
		// Set 写入缓存数据
		Set(ctx context.Context, key string, value any, ttl time.Duration) error

		// Get 获取缓存数据
		Get(ctx context.Context, key string) ([]byte, error)

		// SaveTagKey 将缓存key写入tag
		SaveTagKey(ctx context.Context, tag, key string) error

		// RemoveFromTag 根据缓存tag删除缓存
		RemoveFromTag(ctx context.Context, tag string) error
	}
)

type Cache struct {
	store Store

	// Serializer 序列化
	Serializer Serializer

	// prefix 缓存前缀
	prefix string
}

// New
// @param conf
// @date 2022-07-02 08:09:52
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

// Name
// @date 2022-07-02 08:09:48
func (p *Cache) Name() string {
	return "gorm:cache"
}

// Initialize
// @param tx
// @date 2022-07-02 08:09:47
func (p *Cache) Initialize(tx *gorm.DB) error {
	return tx.Callback().Query().Replace("gorm:query", p.Query)
}

// generateKey
// @param key
// @date 2022-07-02 08:09:46
func generateKey(key string) string {
	hash := fnv.New64a()
	_, _ = hash.Write([]byte(key))

	return strconv.FormatUint(hash.Sum64(), 36)
}

// Query
// @param tx
// @date 2022-07-02 08:09:38
func (p *Cache) Query(tx *gorm.DB) {
	ctx := tx.Statement.Context

	var ttl time.Duration
	var hasTTL bool

	if ttl, hasTTL = FromExpiration(ctx); !hasTTL {
		callbacks.Query(tx)
		return
	}

	var (
		key    string
		hasKey bool
	)

	// 调用 Gorm的方法生产SQL
	callbacks.BuildQuerySQL(tx)

	sql := tx.Statement.SQL.String()
	vars := tx.Statement.Vars

	// 用于将参数填充进sql中，比如将"LIMIT = ?"解析成"LIMIT = 2"。
	sql = logger.ExplainSQL(sql, nil, `"`, vars...)

	// 是否有自定义key
	if key, hasKey = FromKey(ctx); !hasKey {
		key = p.prefix + generateKey(sql)
	}

	// 查询缓存数据

	if err := p.QueryCache(ctx, key, tx.Statement.Dest); err == nil {
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
		return
	}

	if tag, hasTag := FromTag(ctx); hasTag {
		_ = p.store.SaveTagKey(ctx, tag, key)
	}
}

// QueryDB 查询数据库数据
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

// QueryCache 查询缓存数据
// @param ctx
// @param key
// @param dest
func (p *Cache) QueryCache(ctx context.Context, key string, dest any) error {

	values, err := p.store.Get(ctx, key)
	if err != nil {
		return err
	}

	switch dest.(type) {
	case *int64:
		dest = 0
	}
	return p.Serializer.Deserialize(values, dest)
}

// SaveCache 写入缓存数据
func (p *Cache) SaveCache(ctx context.Context, key string, dest any, ttl time.Duration) error {
	values, err := p.Serializer.Serialize(dest)
	if err != nil {
		return err
	}

	return p.store.Set(ctx, key, values, ttl)
}

// RemoveFromTag 根据tag删除缓存数据
// @param ctx
// @param tag
// @date 2022-07-02 08:08:59
func (p *Cache) RemoveFromTag(ctx context.Context, tag string) error {
	return p.store.RemoveFromTag(ctx, tag)
}
