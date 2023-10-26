package service

import (
	"context"
	"fmt"

	"github.com/allan-deng/redis-id-generator/internal/generator"

	log "github.com/sirupsen/logrus"
	"github.com/valyala/fasthttp"
)

type Response struct {
	HttpStatus int
	Body       interface{}
}

// 需要使用 jsom mashal 的字段必须是 可导出的。否则将不会处理
type IdRsp struct {
	Ret    int    `json:"ret"`
	Msg    string `json:"msg"`
	BizTag string `json:"biztag"`
	Id     int64  `json:"id"`
}

func GetIdHandler(ctx context.Context, req *fasthttp.Request) Response {
	values := req.URI().QueryArgs()
	bizTag := values.Peek("biztag")
	if bizTag == nil {
		log.Errorf("url lack of biztag")
		return Response{
			Body: IdRsp{
				Ret: 1,
				Msg: "biz tag param err",
			},
		}
	}

	id, err := generator.IdGen.GetId(ctx, string(bizTag))
	if err != nil {
		log.Errorf("get id failed, biz tag: %v, err: %v.", bizTag, err)
		return Response{
			Body: IdRsp{
				Ret:    2,
				Msg:    fmt.Sprintf("get id failed.%v", err.Error()),
				BizTag: string(bizTag),
				Id:     id,
			},
		}
	}
	log.Debugf("get id succ, biz tag: %v, id: %v.", string(bizTag), id)
	return Response{
		Body: IdRsp{
			Ret:    0,
			Msg:    "succ",
			BizTag: string(bizTag),
			Id:     id,
		},
	}
}
