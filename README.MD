```go
package main

import (
	"context"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/liyuan1125/gorm-cache"
	redis2 "github.com/liyuan1125/gorm-cache/store/redis"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"time"
)

func main() {
	dsn := "root:123456@tcp(127.0.0.1:3306)/gorm?charset=utf8&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	redisClient := redis.NewClient(&redis.Options{Addr: ":6379"})

	cacheConfig := &cache.Config{
		Store:      redis2.NewWithDb(redisClient),
		Serializer: &cache.DefaultJSONSerializer{},
	}

	cachePlugin := cache.New(cacheConfig)

	if err = db.Use(cachePlugin); err != nil {
		fmt.Println(err.Error())
		return
	}

	var username string
	ctx := context.Background()
	ctx = cache.NewExpiration(ctx, time.Hour)

	db.Table("users").WithContext(ctx).Where("id = 1").Limit(1).Pluck("username", &username)
	fmt.Println(username)
	// output gorm

	var nickname string
	ctx2 := context.Background()
	ctx2 = cache.NewExpiration(ctx2, time.Hour)
	ctx2 = cache.NewKey(ctx2, "nickname")

	db.Table("users").WithContext(ctx2).Where("id = 1").Limit(1).Pluck("nickname", &nickname)

	fmt.Println(nickname)
	// output gormwithmysql

	fmt.Println(redisClient.Keys(ctx, "*").Val())
	// output [nickname u4snzyfbv2cr]
}

```