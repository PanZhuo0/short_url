package handler

import (
	"net/http"

	"shorturl/internal/logic"
	"shorturl/internal/svc"
	"shorturl/internal/types"

	"github.com/go-playground/validator/v10"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func GetLongHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.GetLongRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		// 参数校验
		if err := validator.New().StructCtx(r.Context(), &req); err != nil {
			logx.Errorw("validator check failed", logx.LogField{Key: "err", Value: err.Error()})
			httpx.ErrorCtx(r.Context(), w, err) // ErrorCtx writes err into w.
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
