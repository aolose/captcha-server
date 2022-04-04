package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"github.com/steambap/captcha"
	"github.com/valyala/fasthttp"
	"log"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

var server *fasthttp.Server

func getIp(ctx *fasthttp.RequestCtx) string {
	ip := string(ctx.Request.Header.Peek("X-Forwarded-For"))
	if index := strings.IndexByte(ip, ','); index >= 0 {
		ip = ip[0:index]
	}
	ip = strings.TrimSpace(ip)
	if len(ip) > 0 {
		return ip
	}
	ip = strings.TrimSpace(string(ctx.Request.Header.Peek("X-Real-Ip")))
	if len(ip) > 0 {
		return ip
	}
	return ctx.RemoteIP().String()
}

func getHeader(ctx *fasthttp.RequestCtx, key string) string {
	return string(ctx.Request.Header.Peek(key))
}

func getKey(ctx *fasthttp.RequestCtx) int64 {
	k := getHeader(ctx, "x-captcha-key")
	if k == "" {
		k = getHeader(ctx, "X-Captcha-Key")
	}
	i, err := strconv.ParseInt(k, 36, 64)
	if err == nil {
		return i
	}
	return 0
}

func getCode(ctx *fasthttp.RequestCtx) string {
	c := getHeader(ctx, "x-captcha-code")
	if c == "" {
		c = getHeader(ctx, "X-Captcha-Code")
	}
	return c
}

func fmtKey(key int64) string {
	return strconv.FormatInt(key, 36)
}
func code64(code *captcha.Data) string {
	emptyBuff := bytes.NewBuffer(nil)
	_ = code.WriteImage(emptyBuff)
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(emptyBuff.Bytes())
}
func renderCode(ctx *fasthttp.RequestCtx) []byte {
	ip := getIp(ctx)
	key, code := generate(ip)
	if code == nil {
		ctx.SetStatusCode(429)
		return []byte(`{"error":"please try again later."}`)
	}
	k := fmtKey(key)
	img := code64(code)
	log.Printf("[generate info] ip:%s key:%s code:%s\n", ip, k, code.Text)
	return []byte(`{"key":"` + k + `","data":"` + img + `"}`)
}

func forward(ctx *fasthttp.RequestCtx) {
	if cfg.Backend != "" {
		req := fasthttp.AcquireRequest()
		resp := fasthttp.AcquireResponse()
		defer fasthttp.ReleaseRequest(req)   // <- do not forget to release
		defer fasthttp.ReleaseResponse(resp) // <- do not forget to release
		ctx.Request.CopyTo(req)
		req.SetBodyRaw(ctx.PostBody())
		req.SetHostBytes(ctx.Host())
		req.Header.Del("x-captcha-key")
		req.Header.Del("X-Captcha-Key")
		req.Header.Del("x-captcha-code")
		req.Header.Del("X-Captcha-Code")
		cli := fasthttp.HostClient{
			Addr: cfg.Backend,
		}
		req.SetRequestURIBytes(ctx.Request.RequestURI())
		if !cfg.ForwardHost {
			host := cfg.Backend
			u, _ := url.Parse(host)
			if u != nil {
				host = u.Host
			}
			req.Header.SetHost(host)
		}
		log.Printf("[forward] %s %s", req.Host(), req.RequestURI())
		err := cli.Do(req, resp)
		if err != nil {
			ctx.SetStatusCode(500)
			ctx.SetBody([]byte(fmt.Sprintf(`{"error":"%v"}`, err)))
		} else {
			resp.CopyTo(&ctx.Response)
			ctx.SetBody(resp.Body())
		}
	}
}

func loadFont() {
	fd, err := os.ReadFile(cfg.Font)
	if err == nil {
		_ = captcha.LoadFont(fd)
	}
}

func serverHandler(ctx *fasthttp.RequestCtx) {
	n := time.Now().UnixMilli()
	defer func() { log.Printf("%dms\t%s %s\n", time.Now().UnixMilli()-n, ctx.Method(), ctx.Path()) }()
	code := getCode(ctx)
	if code == "" {
		if ctx.Method()[0] == 'G' {
			ctx.SetBody(renderCode(ctx))
			return
		}
	} else {
		key := getKey(ctx)
		if key > 0 {
			nKey, nCode := Check(key, code, getIp(ctx))
			switch nKey {
			case 0:
				forward(ctx)
				return
			case -1:
				ctx.SetStatusCode(429)
				ctx.SetBody([]byte(`{"error":"please try again later."}`))
				return
			default:
				ctx.SetStatusCode(401)
				ctx.SetBody([]byte(`{"key":"` + fmtKey(nKey) + `","data":"` + code64(nCode) + `"}`))
				return
			}
		}
	}
	ctx.SetStatusCode(404)
}

func start() {
	server = &fasthttp.Server{
		MaxConnsPerIP:                 cfg.Conns,
		Handler:                       serverHandler,
		Name:                          "Captcha Service",
		DisableHeaderNamesNormalizing: true,
		ReadTimeout:                   5 * time.Second, // important
		IdleTimeout:                   0,
	}
	log.Printf("Server run at %s", cfg.Addr)
	err := server.ListenAndServe(cfg.Addr)
	if err != nil {
		log.Fatalln(err)
	}
}
func stop() {
	err := server.Shutdown()
	if err != nil {
		log.Fatalln(err)
	}
}
