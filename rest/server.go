package rest

import (
	chat_handler "eraya/rest/handlers/chat"
	order_handler "eraya/rest/handlers/order"
	product_handler "eraya/rest/handlers/product"
	review_handler "eraya/rest/handlers/review"
	settings_handler "eraya/rest/handlers/settings"
	user_handler "eraya/rest/handlers/user"
	wishlist_handler "eraya/rest/handlers/wishlist"
	erayamiddleware "eraya/rest/middlewares"
	"eraya/user"
	"fmt"
	"net/http"
	"os"
	"strconv"

	_ "eraya/docs"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	httpSwagger "github.com/swaggo/http-swagger/v2"
)

type Server struct {
	port            int
	jwtSecret       string
	userSvc         user.Service
	userHandler     *user_handler.Handler
	productHandler  *product_handler.Handler
	orderHandler    *order_handler.Handler
	reviewHandler   *review_handler.Handler
	chatHandler     *chat_handler.Handler
	wishlistHandler *wishlist_handler.Handler
	settingsHandler *settings_handler.Handler
}

func NewServer(
	port int,
	jwtSecret string,
	userSvc user.Service,
	userHandler *user_handler.Handler,
	productHandler *product_handler.Handler,
	orderHandler *order_handler.Handler,
	reviewHandler *review_handler.Handler,
	chatHandler *chat_handler.Handler,
	wishlistHandler *wishlist_handler.Handler,
	settingsHandler *settings_handler.Handler,
) *Server {
	return &Server{
		port:            port,
		jwtSecret:       jwtSecret,
		userSvc:         userSvc,
		userHandler:     userHandler,
		productHandler:  productHandler,
		orderHandler:    orderHandler,
		reviewHandler:   reviewHandler,
		chatHandler:     chatHandler,
		wishlistHandler: wishlistHandler,
		settingsHandler: settingsHandler,
	}
}

func (server *Server) Start() {
	manager := erayamiddleware.NewManager()
	manager.Use(
		erayamiddleware.Cors,
	)

	mux := chi.NewRouter()
	mux.Use(middleware.Recoverer)
	mux.Use(middleware.RealIP)
	mux.Use(middleware.Compress(5))    // Gzip compression
	mux.Use(erayamiddleware.RateLimit) // IP-based Rate Limiting

	// Global Security Headers
	mux.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-XSS-Protection", "1; mode=block")
			if r.TLS != nil {
				w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			}
			next.ServeHTTP(w, r)
		})
	})

	mux.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	mux.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"), // Relative URL works on both HTTP and HTTPS
	))

	mux.Get("/docs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `
			<!DOCTYPE html>
			<html>
			  <head>
				<title>Eraya API Documentation</title>
				<meta charset="utf-8"/>
				<meta name="viewport" content="width=device-width, initial-scale=1">
				<link href="https://fonts.googleapis.com/css?family=Montserrat:300,400,700|Roboto:300,400,700" rel="stylesheet">
				<style>
				  body {
					margin: 0;
					padding: 0;
				  }
				</style>
			  </head>
			  <body>
				<div id="redoc-container"></div>
				<script src="https://cdn.redoc.ly/redoc/latest/bundles/redoc.standalone.js"></script>
				<script>
				  Redoc.init('/swagger/doc.json', {
					theme: {
					  colors: {
						primary: {
						  main: '#0f172a'
						}
					  },
					  typography: {
						fontFamily: 'Roboto, sans-serif',
						headings: {
						  fontFamily: 'Montserrat, sans-serif'
						}
					  }
					}
				  }, document.getElementById('redoc-container'))
				</script>
			  </body>
			</html>
		`)
	})

	mux.Route("/api/v1", func(r chi.Router) {
		user_handler.RegisterRoutes(r, server.userHandler, server.jwtSecret)
		product_handler.RegisterRoutes(r, server.productHandler, server.jwtSecret, server.userSvc)
		order_handler.RegisterRoutes(r, server.orderHandler, server.jwtSecret, server.userSvc)
		review_handler.RegisterRoutes(r, server.reviewHandler, server.jwtSecret, server.userSvc)
		chat_handler.RegisterRoutes(r, server.chatHandler, server.jwtSecret, server.userSvc)
		wishlist_handler.RegisterRoutes(r, server.wishlistHandler, server.jwtSecret, server.userSvc)
		settings_handler.RegisterRoutes(r, server.settingsHandler, server.jwtSecret, server.userSvc)
	})

	wrappedMux := manager.WrapMux(mux, manager.GetGlobalMiddlewares()...)

	address := ":" + strconv.Itoa(server.port)

	fmt.Println("Server running on", address)
	err := http.ListenAndServe(address, wrappedMux)
	if err != nil {
		fmt.Println("Error starting the server", err)
		os.Exit(1)
	}
}
