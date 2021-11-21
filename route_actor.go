package fasthttp

import (
	"errors"
	"fmt"

	"github.com/7vars/leikari"
	"github.com/7vars/leikari/route"
	"github.com/fasthttp/router"
	"github.com/valyala/fasthttp"
)

func fasthttpHandlerFunc(ref leikari.Ref, log leikari.Logger) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		req := NewRequest(ctx)
		res, err := ref.RequestContext(ctx, req)
		if err != nil {
			log.Error(err)
			ctx.SetStatusCode(fasthttp.StatusInternalServerError)
			fmt.Fprint(ctx, "internal server error")
			return
		}
		if response, ok := res.(route.Response); ok {
			if len(response.Header) > 0 {
				for key, value := range response.Header {
					ctx.Response.Header.Set(key, value)
				}
			}

			ctx.SetContentType(response.ContentType())

			ctx.SetStatusCode(response.StatusCode())
			buf, err := response.Decode()
			if err != nil {
				log.Error(err)
				ctx.SetStatusCode(fasthttp.StatusInternalServerError)
				fmt.Fprint(ctx, "internal server error")
			}
			ctx.SetBody(buf)
			return
		}
		log.Error("no response received")
		ctx.SetStatusCode(fasthttp.StatusNotImplemented)
		fmt.Fprint(ctx, "not implemented")
	}
}

type fastHttpRouteHandler interface {
	Handle(string, string, fasthttp.RequestHandler)
	Group(path string) *router.Group
}

type routeActor struct {
	router fastHttpRouteHandler
	def route.Route
	middleware []route.Middleware
}

func newRouteActor(router fastHttpRouteHandler, def route.Route, middleware ...route.Middleware) leikari.Receiver {
	return &routeActor{
		router: router,
		def: def,
		middleware: middleware,
	}
}

func (ra *routeActor) ActorName() string {
	return ra.def.RouteName()
}

func (ra *routeActor) middlewares() []route.Middleware {
	return append(ra.middleware, ra.def.RouteMiddleware()...)
}
 
func (ra *routeActor) PreStart(ctx leikari.ActorContext) error {
	if ra.def.Handle != nil {
		method := "GET"
		if ra.def.Method != "" {
			method = ra.def.Method
		}
		ra.router.Handle(method, ra.def.Path, fasthttpHandlerFunc(ctx.Self(), ctx.Log()))
	}
	
	if len(ra.def.Routes) > 0 {
		subrouter := ra.router.Group(ra.def.Path)
		for _, childRoute := range ra.def.Routes {
			if _, err := ctx.Execute(newRouteActor(subrouter, childRoute, ra.middlewares()...)); err != nil {
				return err
			}
		}
	}

	return nil
}

func (ra *routeActor) Receive(ctx leikari.ActorContext, msg leikari.Message) {
	if ra.def.Handle == nil {
		msg.Reply(errors.New("route handler not defined"))
		return
	}
	if request, ok := msg.Value().(route.Request); ok {
		handle := ra.def.Handle
		for _, mw := range ra.middlewares() {
			handle = mw(handle)
		}

		msg.Reply(handle(request))
		return
	}
	msg.Reply(fmt.Errorf("unkonwn type %T for Request", msg.Value()))
}

func (ra *routeActor) AsyncActor() bool {
	return true
}