package middleware

import "net/http"

type Middleware func(http.Handler) http.Handler

type Manager struct {
	globalMiddlewares []Middleware
} 

func NewManager() *Manager {
	return &Manager {
		globalMiddlewares: make([]Middleware, 0),
	}
}

func (mngr *Manager) Use(middlewares ...Middleware) {
	mngr.globalMiddlewares = append(mngr.globalMiddlewares, middlewares...)
}

func (mngr *Manager) GetGlobalMiddlewares() []Middleware {
	return mngr.globalMiddlewares
}

func (mngr *Manager) With(handler http.Handler, middlewares ...Middleware) http.Handler {
	h := handler

	for _, middleware := range middlewares {
		h = middleware(h)
	}

	return h
}

func (mngr *Manager) WrapMux(handler http.Handler, middlewares ...Middleware) http.Handler {
	h := handler

	for _, middleware := range middlewares {
		h = middleware(h)
	}

	return h
}