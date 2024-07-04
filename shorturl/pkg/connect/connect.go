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
