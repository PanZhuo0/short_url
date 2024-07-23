# 短链接项目

# 项目整体架构
![申请转链接](https://github.com/PanZhuo0/short_url/blob/master/%E8%AF%B7%E6%B1%82%E8%BD%AC%E9%93%BE.png)
Q：如何避免短链接网址出现敏感词
A：使用黑名单机制，在config.go中ShortUrlBlackList设置即可，当出现敏感词时，会自动跳过这个
	这个短链接直接生成下一个短链接，直到合法


![短链接获取长连接](https://github.com/PanZhuo0/short_url/blob/master/%E8%8E%B7%E5%8F%96%E5%8E%9F%E6%9C%AC%E9%93%BE%E6%8E%A5.png)
Q:如何避免短时间内对同一个短链接的大量访问(缓存击穿问题)
A:使用single flight 合并短时间内多个同样的请求

Q：如何避免攻击者用不存在的短链接获取长链接(缓存穿透问题)
A:使用布隆过滤器过滤，确保传入有效的短链接一定能响应，减少不存在短链接的响应可能性，降低缓存压力
## 搭建项目的骨架

1. 建库建表
```sql
CREATE TABLE `sequence` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `stub` varchar(1) NOT NULL,
  `timestamp` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_uniq_stub` (`stub`)
) ENGINE=MyISAM DEFAULT CHARSET=utf8 COMMENT = '序号表';
```

新建长链接短链接映射表:
```sql
CREATE TABLE `short_url_map` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键',
    `create_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `create_by` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '创建者',
    `is_del` tinyint UNSIGNED NOT NULL DEFAULT '0' COMMENT '是否删除：0正常1删除',
    
    `lurl` varchar(2048) DEFAULT NULL COMMENT '长链接',
    `md5` char(32) DEFAULT NULL COMMENT '长链接MD5',
    `surl` varchar(11) DEFAULT NULL COMMENT '短链接',
    PRIMARY KEY (`id`),
    INDEX(`is_del`),
    UNIQUE(`md5`),
    UNIQUE(`surl`)
)ENGINE=INNODB DEFAULT CHARSET=utf8mb4 COMMENT = '长短链映射表';
```

2. 搭建go-zero框架的骨架

编写`api` 文件，使用goctl 命令生成代码

```api
   /* 短链接项目*/
type ConvertRequest {
	LongUrl string `json:""longUrl`
}

type ConvertResponse {
	ShortUrl string `json:"shortUrl"`
}

type GetLongRequest {
	ShortUrl string `path:"shortUrl"`
}

type GetLongResponse {
	LongUrl string `json:"longUrl"`
}

service shortener-api {
	@handler ConvertHandler
	post /convert (ConvertRequest) returns (ConvertResponse)

	@handler GetLongHandler
	get /:shortUrl (GetLongRequest) returns (GetLongResponse)
}
```

```bash
  goctl api go -api shortener.api -dir . 

```

3. 根据数据表生成model层的代码
```bash
  goctl model mysql datasource -url="root:123123@tcp(localhost:3306)/go" -table="short_url_map" -dir="./model"
  goctl model mysql datasource -url="root:123123@tcp(localhost:3306)/go" -table="sequence" -dir="./model"
```

4. 下载项目依赖
```bash
go mod tidy
```

5. 运行
```bash
go run shortener.go
```

## 配置文件的设置
config.go
```go
package config

import "github.com/zeromicro/go-zero/rest"

type Config struct {
	rest.RestConf

	ShortUrlDB

	SequenceDB struct {
		DSN string
	}
}

type ShortUrlDB struct {
	DSN string
}
```


yaml文件
```yaml
Name: shortener-api
Host: 0.0.0.0
Port: 8888

ShortUrlDB:
  DSN: root:123123@tcp(localhost:3306)/go?parseTime=true

SequenceDB:
  DSN: root:123123@tcp(localhost:3306)/go?parseTime=true
```

测试-shortener.go
```go
func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)
	fmt.Printf("load config,%#v\n", c)

	server := rest.MustNewServer(c.RestConf)
	defer server.Stop()

	ctx := svc.NewServiceContext(c)
	handler.RegisterHandlers(server, ctx)

	fmt.Printf("Starting server at %s:%d...\n", c.Host, c.Port)
	server.Start()
}
```


## 长链接的校验
1. go-zero 使用validator
https://github.com/go-playground/validator

download dependency

```bash
go get github.com/go-playground/validator/v10
```

add validate tag to struct that in api file

```api
type ConvertRequest {
	LongUrl string `json:"longUrl" validate:"required"`
}
```

update server layer code `converthandler.go`
```go 
func ConvertHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// analyze request parameter
		var req types.ConvertRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		// Edited:  parameter check
		if err := validator.New().StructCtx(r.Context(), &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			logx.Error("validator check failed", logx.LogField{Key: "err", Value: err.Error()})
			return
		}

		// execute server logic
		l := logic.NewConvertLogic(r.Context(), svcCtx)
		resp, err := l.Convert(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
```

## 检测长链接本身能否连通
这部分放在/pkg/connect中

`connection.go`
```go
package connect

import (
	"net/http"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

var client = &http.Client{
	Transport: &http.Transport{
		DisableKeepAlives: true, //just a connection test,therefore not KeepAlive
	},
	Timeout: 2 * time.Second,
}

/* Get 判断url 是否能请求通过 */
func Get(url string) bool {
	resp, err := client.Get(url)
	if err != nil {
		logx.Errorw("connect client.Get failed", logx.LogField{Key: "err", Value: err.Error()})
		return false
	}
	resp.Body.Close()
	/* 存在的问题issue:如果这是一个3xx 的响应也会被认为是不可达的 */
	return resp.StatusCode == http.StatusOK
}
```

`convertlogic.go`
```go
	// 1.1 长链接URL应该可以访问
	if ok := connect.Get(req.LongUrl); !ok {
		return nil, errors.New("invalid url")
	}
```
## 检查长链接是否已经被转链接过
使用长链接的MD5值去查看是否转链接过,数据库中有对应MD5记录就是有,否则就是没有
将对应代码放在/pkg/md5 目录下

`md5.go`
>attention:这里使用的是自己写的md5包
```go
package md5

import (
	"crypto/md5"
	"encoding/hex"
)

/* Sum 对传入的data参数求md5值 */
func Sum(data []byte) string {
	h := md5.New()
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil)) //32位的16进制数
}

```

这里使用到了数据库查询（通过查询数据库中的MD5记录来找到对应的记录）,要在serviceCtx 中初始化
```go
type ServiceContext struct {
	Config        config.Config
	ShortUrlModel model.ShortUrlMapModel //sohrt_url_map 这张表
}

func NewServiceContext(c config.Config) *ServiceContext {
	conn := sqlx.NewMysql(c.ShortUrlDB.DSN)
	return &ServiceContext{
		Config:        c,
		ShortUrlModel: model.NewShortUrlMapModel(conn),
	}
}
```
## 检查输入的连接是否是短链接
```go
	// 1.4 避免循环转链接(判断输入的链接不能是一个短链接)
	myUrl, err := urltool.GetBasePath(req.LongUrl)
	if err != nil {
		logx.Errorw("urltool.GetBasePath failed", logx.LogField{Key: "lurl", Value: req.LongUrl})
		return nil, err
	}
	_, err = l.svcCtx.ShortUrlModel.FindOneBySurl(context.Background(), sql.NullString{String: myUrl, Valid: true}) //查询一下这个值是否在短链接表中存在
	if err != sqlx.ErrNotFound {
		if err != nil {
			return nil, fmt.Errorf("该链接已经是短链接了")

		}
		logx.Errorw("ShortUrlModel.FindOneBySurl failed", logx.LogField{Key: "err", Value: err.Error()})
		return nil, err
	}
```

## 单元测试
`VSCODE支持生成单元测试的插件` 直接对某个功能对应的函数签名 右键点击`Go:generate unittest for function`就行 在生成的单元测试文件中，增加对应的测试用例即可，当然也可以手动编写


> Reference:https://github.com/smartystreets/goconvey

go get github.com/smartystreets/goconvey
手动编写的connect_test.go -> 使用goconvery 第三方库

```go
package connect

import (
	"testing"

	"github.com/smartystreets/goconvey/convey"
)

func TestGet(t *testing.T) {
	convey.Convey("基础用例", t, func() {
		url := "https://github.com/smartystreets/goconvey?tab=readme-ov-file"
		got := Get(url)
		// assert
		convey.So(got, convey.ShouldEqual, true)
		convey.ShouldBeTrue(got)
	})
	convey.Convey("url不通过的示例", t, func() {
		url := `posts/Go/unit-test`
		got := Get(url)
		// assert
		convey.ShouldBeFalse(got)
	})
}
```

自动生成的urltool_test.go

```go
package urltool

import "testing"

func TestGetBasePath(t *testing.T) {
	type args struct {
		targetUrl string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		// TODO: Add test cases.
		{name: "基本示例", args: args{targetUrl: "https://github.com/go-playground/validator"}, want: "validator", wantErr: false},
		{name: "相对路径url示例", args: args{targetUrl: "/xxxx/t"}, want: "", wantErr: true},
		{name: "空字符串", args: args{targetUrl: ""}, want: "", wantErr: true},
		{name: "带query的url", args: args{targetUrl: "https://www.liwenzhou.com/posts/Go/redis-otel/?a=1&b=2"}, want: "redis-otel", wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetBasePath(tt.args.targetUrl)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetBasePath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetBasePath() = %v, want %v", got, tt.want)
			}
		})
	}
}
```


## 取号器的实现
> 定义一个取号器接口,从而可以用MySQL/Redis/其他实现,该接口中拥有一个Next方法，返回uint64的号码+一个error
以下是MySQL的实现 /pkg/sequence/mysql.go
```go
package sequence

import (
	"database/sql"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

/* 建立MySQL链接，执行REPLACE INTO 语句
REPLACE INTO sequence(stub) values('a')
SELECT LAST_INSERT_ID()
*/

const sqlReplaceIntoStub = `REPLACE INTO sequence(stub)VALUES('a')`

type MySQL struct {
	conn sqlx.SqlConn
}

func NewMySQL(dsn string) *MySQL {
	conn := sqlx.NewMysql(dsn)
	return &MySQL{
		conn: conn,
	}
}

/* Next方法 用于实现取号器取号的操作 */
func (m *MySQL) Next() (seq uint64, err error) {
	// prepare
	var stmt sqlx.StmtSession
	stmt, err = m.conn.Prepare(sqlReplaceIntoStub) //预编译
	if err != nil {
		logx.Errorw("conn.Prepare failed", logx.LogField{Key: "err", Value: err.Error()})
	}
	defer stmt.Close()
	// 执行
	var rest sql.Result
	rest, err = stmt.Exec()
	if err != nil {
		logx.Errorw("stmt.Exec() failed", logx.LogField{Key: "err", Value: err.Error()})
		return
	}
	// 获取插入ID
	var lid int64
	lid, err = rest.LastInsertId()
	if err != nil {
		logx.Errorw("rest.LastInsertId failed", logx.LogField{Key: "err", Value: err.Error()})
		return
	}
	return uint64(lid), nil
}

```
## 62进制转链
62个字符分别为:0-9 a-z A-Z
/pkg/base62/base62.go 

```go
package base62

import (
	"math"
	"slices"
	"strings"
)

const base62str = `0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ`

/* 把uint64的数字转换位62进制的string */
func Int2String(seq uint64) string {
	ret := make([]byte, 0)
	if seq == 0 {
		return "0"
	}
	for seq > 0 {
		ret = append(ret, base62str[seq%62])
		seq = seq / 62
	}
	slices.Reverse(ret)
	return string(ret)
}

/* String2Int 把62进制的string转换为uint64的数字 */
func String2Int(s string) (seq uint64) {
	bs := []byte(s)
	slices.Reverse(bs)
	for idx, b := range bs {
		base := math.Pow(62, float64(idx))
		seq += uint64(strings.Index(base62str, string(b))) * uint64(base)
	}
	return seq
}

```
## 短链接安全性
> 使用配置文件来配置62位字符的排列顺序，而不是使用有规律的序列,提高安全性

config.go
```go
package config

import "github.com/zeromicro/go-zero/rest"

type Config struct {
	rest.RestConf

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

```

xxx.yaml

```yaml
Name: shortener-api
Host: 0.0.0.0
Port: 8888


ShortUrlDB:
  DSN: root:123123@tcp(localhost:3306)/go?parseTime=true

SequenceDB:
  DSN: root:123123@tcp(localhost:3306)/go?parseTime=true


# 需要修改这个
Base62String: 0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ

ShortUrlBlackList: ["version","fuck","stupid","convert","health","api","css","js"]

ShortDomain: "test.cn"

```

## 短链接黑名单
> 需要避免一些侮辱性的短链接，以及某些内部特殊含义的短链接比如health、version等

在配置文件中设置黑名单
xxx.yaml

```yaml
# 设置黑名单
ShortUrlBlackList: ["version","fuck","stupid","convert","health","api","css","js"]
```

> 为了提高处理的速度，程序内建立HashMap来实现O（1）时间复杂度的检索

serverCtx.go

```go
	package svc

import (
	"shorturl/internal/config"
	"shorturl/model"
	"shorturl/sequence"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type ServiceContext struct {
	Config        config.Config
	ShortUrlModel model.ShortUrlMapModel //sohrt_url_map 这张表

	Sequence         sequence.Sequence
	ShortUrlBlackMap map[string]struct{}
}

func NewServiceContext(c config.Config) *ServiceContext {
	conn := sqlx.NewMysql(c.ShortUrlDB.DSN)
	m := make(map[string]struct{}, len(c.ShortUrlBlackList))
	// 根据配置文件中的ShortUrlBlackList 标记黑名单HashMap
	for _, v := range c.ShortUrlBlackList {
		m[v] = struct{}{}
	}

	return &ServiceContext{
		Config:        c,
		ShortUrlModel: model.NewShortUrlMapModel(conn),
		Sequence:      sequence.NewMySQL(c.SequenceDB.DSN), //sequence 表
		// Sequence:      sequence.NewRedis(redisAddr), //redis
		ShortUrlBlackMap: m,
	}
}
```

实现黑名单控制逻辑  convertlogic.go

```go

	var short string
	for {
		// 2.转链接(用发号器产生短链接的地址)
		/* 每来一个长链接，我们就使用一个REPLACE INTO 语句往sequence 表插入一条数据，并且取出主键ID作为号码*/
		seq, err := l.svcCtx.Sequence.Next()
		if err != nil {
			logx.Errorw("Sequence.Next() failed", logx.LogField{Key: "err", Value: err.Error()})
			return nil, err
		}
		fmt.Println(seq)
		// 3.号码装入短链接
		/*
			3.1安全性问题
			为了避免恶意请求，可以把62位字符次序打乱
		*/

		/*
			3.2短域名避免某些特殊词
			比如fuck\stupid\asshole\
			或者api\version\health
			需要建立一个黑名单机制，避免某些特殊的词的出现
		*/
		short = base62.Int2String(seq)
		if _, ok := l.svcCtx.ShortUrlBlackMap[short]; !ok {
			break
		}
	}

```


## 查看短链接
getlonglogic.go

```go

func (l *GetLongLogic) GetLong(req *types.GetLongRequest) (resp *types.GetLongResponse, err error) {
	// todo: add your logic here and delete this line
	/*
		1.根据短链接查询长链接
		2.返回302响应
	*/
	u, err := l.svcCtx.ShortUrlModel.FindOneBySurl(l.ctx, sql.NullString{String: req.ShortUrl, Valid: true})
	if err != nil {
		if err == sql.ErrNoRows {
			// 没有查到数据
			return nil, errors.New("404")
		}
		logx.Errorw("ShortUrlModel.FindOneBySurl failed", logx.LogField{Value: err.Error(), Key: "err"})
		return nil, err
	}
	return &types.GetLongResponse{LongUrl: u.Lurl.String}, nil
}

```

处理重定向的handler getlonghandler.go
```go

func GetLongHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.GetLongRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := logic.NewGetLongLogic(r.Context(), svcCtx)
		resp, err := l.GetLong(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			// httpx.OkJsonCtx(r.Context(), w, resp)
			// 这里应该返回重定向的响应
			http.Redirect(w, r, resp.LongUrl, http.StatusFound) //302
		}
	}
}

```

> 存在的问题:请求量过大时 eg:10000 次短链接请求/s,那么数据库一秒就要被查询10000次(数据库还会有别的业务的查询、修改操作)，这会导致响应速度降低 应该使用缓存加速

## 查看短链接+缓存
自己写缓存			surl-> lurl
go-zero自带的缓存 	surl-> 数据行(信息冗余),代码量小,不需要自己实现

这里使用go-zero自带的

Q:如何在go-zero中增加缓存层?
1. 修改config.go
```go
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
```
2. 修改.yaml
```yaml
CacheRedis: 
  - Host: 127.0.0.1:6379

```

##### 实现缓存层
1. 删除原有的model层代码
	- 删除shorturlmapmodel.go

2. 重新生成model层代码
```bash
	goctl model mysql datasource -url="root:123123@tcp(127.0.0.1:3306)/go" -table="short_url_map" -dir="./model" -c 
```

3. 修改servicecontext.go代码 (增加一个redis的初始化)
```go
	func NewServiceContext(c config.Config) *ServiceContext {
	conn := sqlx.NewMysql(c.ShortUrlDB.DSN)
	m := make(map[string]struct{}, len(c.ShortUrlBlackList))
	// 根据配置文件中的ShortUrlBlackList 标记黑名单HashMap
	for _, v := range c.ShortUrlBlackList {
		m[v] = struct{}{}
	}

	return &ServiceContext{
		Config:        c,
		ShortUrlModel: model.NewShortUrlMapModel(conn, c.CacheRedis),
		Sequence:      sequence.NewMySQL(c.SequenceDB.DSN), //sequence 表
		// Sequence:      sequence.NewRedis(redisAddr), //redis
		ShortUrlBlackMap: m,
	}
}
```

> 这样之后，请求将会先走缓存，如果缓存中没有则去MySQL查询，go-zero帮助我们实现了查询的缓存层与数据库层的处理，提高了效率，当然同时也丢失了一定的可见性

go-zero中缓存层的具体实现代码
```go 
func (m *defaultShortUrlMapModel) FindOneBySurl(ctx context.Context, surl sql.NullString) (*ShortUrlMap, error) {
	goShortUrlMapSurlKey := fmt.Sprintf("%s%v", cacheGoShortUrlMapSurlPrefix, surl)
	var resp ShortUrlMap
	err := m.QueryRowIndexCtx(ctx, &resp, goShortUrlMapSurlKey, m.formatPrimary, func(ctx context.Context, conn sqlx.SqlConn, v any) (i any, e error) {
		query := fmt.Sprintf("select %s from %s where `surl` = ? limit 1", shortUrlMapRows, m.table)
		if err := conn.QueryRowCtx(ctx, &resp, query, surl); err != nil {
			return nil, err
		}
		return resp.Id, nil
	}, m.queryPrimary)
	switch err {
	case nil:
		return &resp, nil
	case sqlc.ErrNotFound:
		return nil, ErrNotFound
	default:
		return nil, err
	}
}
```

## 使用缓存后带来的问题
1. 缓存如何设置,淘汰算法LRU(last rencently use)
	1.Redis集群部署
	2.根据数据量设置内存大小，
2. 如何解决缓存击穿问题？(突然大量请求一个缓存失效的项)
	1.过期时间设置大
	2.加锁
	3.使用singleflght 合并并发的请求

3. 如何解决缓存穿透问题 (循环请求来刷新掉缓存中的数据,导致所有请求都走数据库)
	1.布隆过滤器(易于实现)
	2.布谷鸟过滤器(支持删除)

## 使用singleflght 解决缓存击穿问题
> Reference https://pkg.go.dev/golang.org/x/sync/singleflight

`singleflight`提供的功能:重复函数调用抑制机制
	如果请求中第一个调用未完成，后续的重复调用将会保持等待，当第一个调用完成时则会与它们共享结果，这样以来只需要一次函数调用,所有调用都会拿到最终的调用结果
	不同于pipeline，这个不是将请求打包，而是一个请求得到多个请求的结果

singleflight的Do方法

```go

// Do executes and returns the results of the given function, making
// sure that only one execution is in-flight for a given key at a
// time. If a duplicate comes in, the duplicate caller waits for the
// original to complete and receives the same results.
// The return value shared indicates whether v was given to multiple callers.
func (g *Group) Do(key string, fn func() (interface{}, error)) (v interface{}, err error, shared bool) {}

```

DoChan方法

```go

// DoChan is like Do but returns a channel that will receive the
// results when they are ready.
//
// The returned channel will not be closed.
func (g *Group) DoChan(key string, fn func() (interface{}, error)) <-chan Result {}

```

Forget方法
```go
// Forget tells the singleflight to forget about a key.  Future calls
// to Do for this key will call the function rather than waiting for
// an earlier call to complete.
func (g *Group) Forget(key string) {
	g.mu.Lock()
	delete(g.m, key)
	g.mu.Unlock()
}
```

> singleflight的应用场景：防止缓存击穿  | 通过强制一个函数的所有后继调用等待第一个调用完成，消除了同时运行重复函数的低效性

*go-zero缓存本身支持singleflight*
	如果配置的是单节点，缓存会直接进行singleflight类似的操作
	如果是redis集群，会通过一致性hash将请求发完对应的一个Redis节点，然后执行singleflight类似的操作


go-zero中实现singleflight特性的部分代码
```go
func (c cacheNode) doTake(ctx context.Context, v any, key string,
	query func(v any) error, cacheVal func(v any) error) error {
	logger := logx.WithContext(ctx)
	val, fresh, err := c.barrier.DoEx(key, func() (any, error) {
		if err := c.doGetCache(ctx, key, v); err != nil {
			if errors.Is(err, errPlaceholder) {
				return nil, c.errNotFound
			} else if !errors.Is(err, c.errNotFound) {
				// why we just return the error instead of query from db,
				// because we don't allow the disaster pass to the dbs.
				// fail fast, in case we bring down the dbs.
				return nil, err
			}

			if err = query(v); errors.Is(err, c.errNotFound) {
				if err = c.setCacheWithNotFound(ctx, key); err != nil {
					logger.Error(err)
				}

				return nil, c.errNotFound
			} else if err != nil {
				c.stat.IncrementDbFails()
				return nil, err
			}

			if err = cacheVal(v); err != nil {
				logger.Error(err)
			}
		}

		return jsonx.Marshal(v)
	})
	if err != nil {
		return err
	}
	if fresh {
		return nil
	}

	// got the result from previous ongoing query.
	// why not call IncrementTotal at the beginning of this function?
	// because a shared error is returned, and we don't want to count.
	// for example, if the db is down, the query will be failed, we count
	// the shared errors with one db failure.
	c.stat.IncrementTotal()
	c.stat.IncrementHit()

	return jsonx.Unmarshal(val.([]byte), v)
}
```

```go
package syncx

import "sync"

type (
	// SingleFlight lets the concurrent calls with the same key to share the call result.
	// For example, A called F, before it's done, B called F. Then B would not execute F,
	// and shared the result returned by F which called by A.
	// The calls with the same key are dependent, concurrent calls share the returned values.
	// A ------->calls F with key<------------------->returns val
	// B --------------------->calls F with key------>returns val
	SingleFlight interface {
		Do(key string, fn func() (any, error)) (any, error)
		DoEx(key string, fn func() (any, error)) (any, bool, error)
	}

	call struct {
		wg  sync.WaitGroup
		val any
		err error
	}

	flightGroup struct {
		calls map[string]*call
		lock  sync.Mutex
	}
)

// NewSingleFlight returns a SingleFlight.
func NewSingleFlight() SingleFlight {
	return &flightGroup{
		calls: make(map[string]*call),
	}
}

func (g *flightGroup) Do(key string, fn func() (any, error)) (any, error) {
	c, done := g.createCall(key)
	if done {
		return c.val, c.err
	}

	g.makeCall(c, key, fn)
	return c.val, c.err
}

func (g *flightGroup) DoEx(key string, fn func() (any, error)) (val any, fresh bool, err error) {
	c, done := g.createCall(key)
	if done {
		return c.val, false, c.err
	}

	g.makeCall(c, key, fn)
	return c.val, true, c.err
}

func (g *flightGroup) createCall(key string) (c *call, done bool) {
	g.lock.Lock()
	if c, ok := g.calls[key]; ok {
		g.lock.Unlock()
		c.wg.Wait()
		return c, true
	}

	c = new(call)
	c.wg.Add(1)
	g.calls[key] = c
	g.lock.Unlock()

	return c, false
}

func (g *flightGroup) makeCall(c *call, key string, fn func() (any, error)) {
	defer func() {
		g.lock.Lock()
		delete(g.calls, key)
		g.lock.Unlock()
		c.wg.Done()
	}()

	c.val, c.err = fn()
}

```

## 使用布隆过滤器 解决缓存穿透问题
Reference 
> https://www.jasondavies.com/bloomfilter/ 布隆过滤器可视化

> https://en.wikipedia.org/wiki/Bloom_filter 百科

> https://pkg.go.dev/github.com/devopsfaith/bloomfilter go实现布隆过滤器

> https://github.com/zeromicro/go-zero/blob/master/core/bloom/bloom.go go-zero实现布隆过滤器

应用
	1.防止缓存击穿
	2.推荐系统去重
	3.黑白名单
	4.垃圾邮件过滤

在该项目中使用布隆过滤器
1. 在svcCtx中增加布隆过滤器

```go
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
```

2. 在生成短链后将对应记录记录到布隆过滤器中

convertlogic.go

```go
	// 4.2 将生成的短链接增加到布隆过滤器中
	if err := l.svcCtx.Filter.Add([]byte(short)); err != nil {
		logx.Errorw("BloomFilter.Add() failed", logx.LogField{Key: "err", Value: err.Error()})
	}
```

> 发现错误: 之前已经转过短链接的链接无法使用，因为那些记录没有映射到布隆过滤器中(向前兼容问题)
3. 将数据库中已有的映射记录增加到布隆过滤器中(更新)
```go
	func loadDataTobloomFilter(){} //该函数用于将内存中的布隆过滤器数据保存到数据库中，也就是持久化
	// 数据可能过大,需采用分页查询实现查询数据,并加载到内存
```

## 扩展-布谷鸟过滤器
reference
> https://www.lkozma.net/cuckoo_hashing_visualization/ 可视化

> https://www.brics.dk/RS/01/32/BRICS-RS-01-32.pdf 论文

## 短链接项目总结
1. 项目的逻辑图
2. 项目的回顾
3. 个人收获
	Todo:

## 对项目的可能的扩展
1. 如何支持自定义的短链接？
	维护一个已经使用的序号，后续生成序号时判断是否已经被分配

2. 如何让短链接支持过期时间?
	每个链接映射额外记录一个过期时间字段,到期后该映射记录删除。
	关于删除的策略有以下几种:
		1.延迟删除,每次请求时判断是否过期，如果过期则删除
			实现简单，性能损耗小
			存储空间的利用效率底，已经过期的数据可能永远不会被删除

		2.定时删除:创建记录时根据过期时间设置定时器
			过期数据能被及时删除，存储空间的利用率高
			占用内存大，性能差

		3.轮询删除:通过异步脚本在业务低峰期周期性扫表清理过期数据
			兼顾效率和磁盘利用率

3. 如何提高吞吐量
	将整个系统分为写(生成短链接)和读(访问短链接)两个部分
	1.水平扩展多个节点，根据需要分片partition

4. 延迟优化
	整个系统分为写(生成短链接)和读(访问短链接)两个部分
	1. 存储层
		数据结构简单可以直接改用KV存储(或Redis存储)
		对数据节点进行分片，减少单点的工作压力

	2. 缓存层
		增加缓存层，本地缓存---> redis缓存
		增加布隆过滤器判断长链接映射是否已经存在，判断短链接是否有效

	3. 网络
		基于地理位置就近访问数据节点

