package svc

import (
	"shorturl/internal/config"
	"shorturl/model"
	"shorturl/sequence"

	"github.com/zeromicro/go-zero/core/bloom"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type ServiceContext struct {
	Config        config.Config
	ShortUrlModel model.ShortUrlMapModel //sohrt_url_map 这张表

	Sequence         sequence.Sequence
	ShortUrlBlackMap map[string]struct{}
	Filter           *bloom.Filter
}

func NewServiceContext(c config.Config) *ServiceContext {
	conn := sqlx.NewMysql(c.ShortUrlDB.DSN)
	m := make(map[string]struct{}, len(c.ShortUrlBlackList))
	// 根据配置文件中的ShortUrlBlackList 标记黑名单HashMap
	for _, v := range c.ShortUrlBlackList {
		m[v] = struct{}{}
	}
	// 增加布隆过滤器
	store := redis.New(c.CacheRedis[0].Host, func(r *redis.Redis) {
		r.Type = redis.NodeType
	})
	filter := bloom.New(store, "bloom_filter", 20*(1<<20))

	return &ServiceContext{
		Config:        c,
		ShortUrlModel: model.NewShortUrlMapModel(conn, c.CacheRedis),
		Sequence:      sequence.NewMySQL(c.SequenceDB.DSN), //sequence 表
		// Sequence:      sequence.NewRedis(redisAddr), //redis
		ShortUrlBlackMap: m,
		Filter:           filter,
	}
}
