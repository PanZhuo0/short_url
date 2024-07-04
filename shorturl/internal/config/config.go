package config

import (
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/rest"
)

type Config struct {
	rest.RestConf
	CacheRedis cache.CacheConf //redis 缓存

	/* 短链接对应的域名 */
	ShortDomain string
	/* 短链接数据库 */
	ShortUrlDB ShortUrlDB

	/* 发号器数据库 */
	SequenceDB struct {
		DSN string
	}

	/* 用于转链接的62位字符串(尽量乱序) */
	Base62String string

	/* 短链接黑名单 */
	ShortUrlBlackList []string
}

type ShortUrlDB struct {
	DSN string
}
