package logic

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"shorturl/internal/svc"
	"shorturl/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetLongLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetLongLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetLongLogic {
	return &GetLongLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

var Err404 = errors.New("404")

func (l *GetLongLogic) GetLong(req *types.GetLongRequest) (resp *types.GetLongResponse, err error) {
	// todo: add your logic here and delete this line
	/*
		1.根据短链接查询长链接
		2.返回302响应
	*/
	/* 使用布隆过滤器解决缓存穿透问题:
	可选的方案:
		1.基于内存实现(如果服务重启，过滤器中的记录将被删除)
		2.基于Redis实现（由于Redis支持持久化，过滤器中的记录不会被删除）
	*/
	/*
		方案一:基于内存实现
			1.去svcContext中初始化一个go-布隆过滤器
			2.每次重启的时候，都需要去加载已有的短链接数据
			3.定时对短链接布隆过滤器数据进行保存(定时任务)

			func loadDataTobloomFilter(){} //该函数用于将内存中的布隆过滤器数据保存到数据库中，也就是持久化
	*/

	/*
		方案二：基于Redis实现
			todo: add your redis implement here and delete this line
	*/

	exist, err := l.svcCtx.Filter.Exists([]byte(req.ShortUrl)) //查看这个短URL是否在数据库中(可能误判为存在)
	if err != nil {
		logx.Errorw("Bloom filter failed", logx.LogField{Value: err.Error(), Key: "err"})
	}

	// 不存在短链接则直接返回
	if !exist {
		return nil, Err404
	}

	// 可能存在，去查询数据层
	/*
		SingleFlight:
			解决缓存击穿问题(同时有大量请求不在缓存中的数据)(eg:同时有100k个请求不在缓存的记录的request)
			方法:
				合并大量并发的请求
				第一个请求先去请求DB，并发的后99999个请求将会等待第一个请求的结果，将这个结果当成自己的结果

			底层实现:(?)
	*/
	fmt.Println("开始查询缓存和DB...")
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
