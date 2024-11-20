package mux

import (
	"log"
	"net/http"
	"strings"
)

// MiddlewareFunc is a function which receives an http.Handler and returns another http.Handler.
// Typically, the returned handler is a closure which does something with the http.ResponseWriter and http.Request passed
// to it, and then calls the handler passed as parameter to the MiddlewareFunc.
type MiddlewareFunc func(http.Handler) http.Handler

// MiddlewareFuncWithLogging is a middleware function with optional logging.
// It wraps a MiddlewareFunc and adds logging capabilities.
type MiddlewareFuncWithLogging struct {
	Handler MiddlewareFunc
	Name    string
}

// middleware interface is anything which implements a MiddlewareFunc named Middleware.
type middleware interface {
	Middleware(handler http.Handler) http.Handler
}

// Middleware allows MiddlewareFunc to implement the middleware interface.
func (mw MiddlewareFunc) Middleware(handler http.Handler) http.Handler {
	return mw(handler)
}

// Middleware allows MiddlewareFuncWithLogging to implement the middleware interface.
func (mw MiddlewareFuncWithLogging) Middleware(handler http.Handler) http.Handler {
	return mw.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Executing middleware: %s", mw.Name)
		handler.ServeHTTP(w, r)
	}))
}

// Use appends a MiddlewareFunc to the chain. Middleware can be used to intercept or otherwise modify requests and/or responses, and are executed in the order that they are applied to the Router.
func (r *Router) Use(mwf ...MiddlewareFunc) {
	for _, fn := range mwf {
		r.middlewares = append(r.middlewares, fn)
	}
}

// UseWithLogging appends a MiddlewareFuncWithLogging to the chain, allowing optional logging.
func (r *Router) UseWithLogging(name string, mw MiddlewareFunc) {
	r.useInterface(MiddlewareFuncWithLogging{
		Handler: mw,
		Name:    name,
	})
}

// useInterface appends a middleware to the chain. Middleware can be used to intercept or otherwise modify requests and/or responses, and are executed in the order that they are applied to the Router.
func (r *Router) useInterface(mw middleware) {
	r.middlewares = append(r.middlewares, mw)
}

// RouteMiddleware -------------------------------------------------------------

// Use appends a MiddlewareFunc to the chain. Middleware can be used to intercept or otherwise modify requests and/or responses, and are executed in the order that they are applied to the Route. Route middleware are executed after the Router middleware but before the Route handler.
func (r *Route) Use(mwf ...MiddlewareFunc) *Route {
	for _, fn := range mwf {
		r.middlewares = append(r.middlewares, fn)
	}

	return r
}

// UseWithLogging appends a MiddlewareFuncWithLogging to the route's middleware chain.
func (r *Route) UseWithLogging(name string, mw MiddlewareFunc) *Route {
	r.useInterface(MiddlewareFuncWithLogging{
		Handler: mw,
		Name:    name,
	})

	return r
}

// useInterface appends a middleware to the chain. Middleware can be used to intercept or otherwise modify requests and/or responses, and are executed in the order that they are applied to the Route. Route middleware are executed after the Router middleware but before the Route handler.
func (r *Route) useInterface(mw middleware) {
	r.middlewares = append(r.middlewares, mw)
}

// CORSMethodMiddleware automatically sets the Access-Control-Allow-Methods response header
// on requests for routes that have an OPTIONS method matcher to all the method matchers on
// the route. Routes that do not explicitly handle OPTIONS requests will not be processed
// by the middleware. See examples for usage.
func CORSMethodMiddleware(r *Router) MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			allMethods, err := getAllMethodsForRoute(r, req)
			if err == nil {
				for _, v := range allMethods {
					if v == http.MethodOptions {
						w.Header().Set("Access-Control-Allow-Methods", strings.Join(allMethods, ","))
					}
				}
			}

			next.ServeHTTP(w, req)
		})
	}
}

// getAllMethodsForRoute returns all the methods from method matchers matching a given
// request.
func getAllMethodsForRoute(r *Router, req *http.Request) ([]string, error) {
	var allMethods []string

	for _, route := range r.routes {
		var match RouteMatch
		if route.Match(req, &match) || match.MatchErr == ErrMethodMismatch {
			methods, err := route.GetMethods()
			if err != nil {
				return nil, err
			}

			allMethods = append(allMethods, methods...)
		}
	}

	return allMethods, nil
}
