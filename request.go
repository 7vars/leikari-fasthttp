package fasthttp

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/url"

	"github.com/7vars/leikari/route"
	"github.com/valyala/fasthttp"
)

type request struct {
	req *fasthttp.RequestCtx
}

func NewRequest(r *fasthttp.RequestCtx) route.Request {
	return &request{
		req: r,
	}
}

func (r *request) Context() context.Context {
	return r.req
}

func (r *request) URL() *url.URL {
	url, _ := url.Parse(r.req.URI().String())
	return url
}

func (r *request) GetHeader(key string) string {
	buf := r.req.Request.Header.Peek(key)
	return string(buf)
}

func (r *request) GetVar(key string) string {
	val := r.req.UserValue(key)
	if val != nil {
		return fmt.Sprintf("%v", val)
	}
	return ""
}

func (r *request) Body() ([]byte, error) {
	return r.req.Request.Body(), nil
}

func (r *request) Encode(v interface{}) error {
	switch r.GetHeader("Content-Type") {
	case "application/xml":
		return r.Unmarshal(v, xml.Unmarshal)
	// TODO other content-types here
	default:
		return r.Unmarshal(v, json.Unmarshal)
	}
}

func (r *request) Unmarshal(v interface{}, f func([]byte, interface{}) error) error {
	body, err := r.Body()
	if err != nil {
		return err
	}
	return f(body, v)
}