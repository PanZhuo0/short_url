package logic

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"shorturl/internal/svc"
	"shorturl/internal/types"
	"shorturl/model"
	"shorturl/pkg/base62"
	"shorturl/pkg/connect"
	"shorturl/pkg/md5"
	"shorturl/pkg/urltool"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type ConvertLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewConvertLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ConvertLogic {
	return &ConvertLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

/*
	 Convert 转链接
		转链接:输入一个长链接->转为短链接
*/
func (l *ConvertLogic) Convert(req *types.ConvertRequest) (resp *types.ConvertResponse, err error) {
	// todo: add your logic here and delete this line
	// 1.校验数据
	// 1.1 长链接URL应该可以访问
	if ok := connect.Get(req.LongUrl); !ok {
		return nil, errors.New("invalid url")
	}
	// 1.2 数据不能为空(用validator tag)
	// 1.3 判断这个长链接是否已经转链接过了?(增加了缓存层)
	/* 先去Redis查是否有该记录,再去数据库中查询这个MD5记录是否存在，存在就说明已经做过转链接了 */
	md5Value := md5.Sum([]byte(req.LongUrl)) //Attention:这里使用的是自己写的md5包
	u, err := l.svcCtx.ShortUrlModel.FindOneByMd5(context.Background(), sql.NullString{String: md5Value, Valid: true})
	if err != sqlx.ErrNotFound {
		if err == nil {
			return nil, fmt.Errorf("该链接已被转为%s", u.Surl.String)
		}
		/* 出现了意料之外的错误 */
		logx.Errorw("ShortUrlModel.FindOneByMd5 failed", logx.LogField{Key: "err", Value: err.Error()})
		return nil, err
	}
	/* 没找到这个短链接,需要convert,进入下一步判断是否为短链接*/
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
		logx.Errorw("ShortUrlModel.FindOneBySurl failed", logx.LogField{Key: "err", Value: err})
		return nil, err
	}

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
	// 4.1 保存起来(把短链接和长链接的映射关系保存到数据库)
	if _, err := l.svcCtx.ShortUrlModel.Insert(
		l.ctx,
		&model.ShortUrlMap{
			Lurl: sql.NullString{String: req.LongUrl, Valid: true},
			Md5:  sql.NullString{String: md5Value, Valid: true},
			Surl: sql.NullString{String: short, Valid: true},
		},
	); err != nil {
		logx.Errorw("ShortUrlModel.Insert failed", logx.LogField{Key: "err", Value: err.Error()})
		return nil, err
	}
	// 4.2 将生成的短链接增加到布隆过滤器中
	if err := l.svcCtx.Filter.Add([]byte(short)); err != nil {
		logx.Errorw("BloomFilter.Add() failed", logx.LogField{Key: "err", Value: err.Error()})
	}
	// 5.返回响应
	/* 记得增加短域名+短链接 */
	shortUrl := l.svcCtx.Config.ShortDomain + "/" + short
	return &types.ConvertResponse{ShortUrl: shortUrl}, nil
}
