package router

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"runtime/debug"
	"sync"
	"time"
	"unsafe"

	"github.com/allan-deng/redis-id-generator/internal/service"

	"github.com/buaazp/fasthttprouter"
	log "github.com/sirupsen/logrus"
	"github.com/valyala/fasthttp"
)

var svrRouter *fasthttprouter.Router
var once sync.Once
var filterList []Filter

type Filter func(h fasthttp.RequestHandler) fasthttp.RequestHandler

type Handler func(ctx context.Context, req *fasthttp.Request) service.Response

const (
	NONEMETHOD = 0 + iota
	GETMETHOD
	POSTMETHOD
	MAXMETHOD
)

func GetRouter() *fasthttprouter.Router {
	once.Do(func() {
		svrRouter = fasthttprouter.New()
		filterList = make([]Filter, 0)
	})

	if svrRouter == nil {
		panic("router not init!")
	}
	AddFilter(recoverFilter)
	AddFilter(debugLogFilter)
	RegisterHander(GETMETHOD, "/id", service.GetIdHandler)

	return svrRouter
}

func RegisterHander(method int, path string, handler Handler) {
	if method <= NONEMETHOD || method >= MAXMETHOD {
		panic(fmt.Sprintf("register unsupport method %v, path %v!", method, path))
	}
	wrappedHandler := handlerWrapper(handler)
	hanflerChains := processFilter(wrappedHandler, len(filterList)-1)

	if method == GETMETHOD {
		svrRouter.GET(path, hanflerChains)
	} else {
		svrRouter.POST(path, hanflerChains)
	}
}

func AddFilter(filter Filter) {
	filterList = append(filterList, filter)
}

func processFilter(h fasthttp.RequestHandler, index int) fasthttp.RequestHandler {
	if index == -1 {
		return h
	}
	fWrap := filterList[index](h)
	index--
	return processFilter(fWrap, index)
}

func handlerWrapper(h Handler) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		resp := h(ctx, &ctx.Request)
		httpStatus := resp.HttpStatus
		if httpStatus == 0 {
			httpStatus = 200
		}

		body, err := json.Marshal(resp.Body)
		var str string
		if err != nil || resp.Body == nil {
			str = "{\"ret\":999,\"msg\":\"resp json marshal failed\"}"
		} else {
			str = byte2str(body)
		}
		ctx.WriteString(str)
		ctx.SetStatusCode(httpStatus)
	}
}

func recoverFilter(h fasthttp.RequestHandler) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		defer func() {
			if r := recover(); r != nil {
				log.Errorf("Panic occurred: %v, stack: %v", r, debug.Stack())
				ctx.WriteString("{\"ret\":999,\"msg\":\"panic occurred\"}")
				ctx.SetStatusCode(500)
			}
		}()
		h(ctx)
	}

}

func debugLogFilter(h fasthttp.RequestHandler) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		start := time.Now()
		h(ctx)
		cost := time.Since(start)

		log.Debugf("http_status: %d, cost: %dus, req: %s?%s, rsp: %s.", ctx.Response.StatusCode(), cost.Nanoseconds()/int64(time.Microsecond), ctx.Request.URI().Path(), ctx.Request.URI().QueryArgs(), ctx.Response.Body())
	}
}

func byte2str(bytes []byte) string {
	bytePointer := unsafe.Pointer(&bytes)
	byteHeader := (*reflect.SliceHeader)(bytePointer)
	strHeader := reflect.StringHeader{
		Data: byteHeader.Data,
		Len:  byteHeader.Len,
	}
	return *(*string)(unsafe.Pointer(&strHeader))
}
