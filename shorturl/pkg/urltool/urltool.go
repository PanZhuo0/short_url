package urltool

import (
	"errors"
	"net/url"
	"path"

	"github.com/zeromicro/go-zero/core/logx"
)

func GetBasePath(targetUrl string) (string, error) {
	myUrl, err := url.Parse(targetUrl)
	if err != nil {
		logx.Errorw("url.Parse failed", logx.LogField{Key: "err", Value: err.Error()})
		return "", err
	}
	if len(myUrl.Host) == 0 {
		return "", errors.New("no host in target url")
	}
	return path.Base(myUrl.Path), nil
}
