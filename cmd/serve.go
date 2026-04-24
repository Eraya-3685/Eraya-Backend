package cmd

import (
	chat_pkg "eraya/chat"
	"eraya/config"
	"eraya/infra/db"
	"eraya/infra/redis"
	"eraya/infra/storage"
	"eraya/order"
	"eraya/product"
	"eraya/repo"
	"eraya/rest"
	chat_handler "eraya/rest/handlers/chat"
	order_handler "eraya/rest/handlers/order"
	product_handler "eraya/rest/handlers/product"
	review_handler "eraya/rest/handlers/review"
	user_handler "eraya/rest/handlers/user"
	"eraya/review"
	"eraya/user"
	"log"
	"log/slog"

	"net/http"
	"os"
	"time"
)

func keepAlive(url string) {
	ticker := time.NewTicker(14 * time.Minute)
	for range ticker.C {
		resp, err := http.Get(url)
		if err == nil {
			log.Printf("Keep-alive ping sent to %s | Status: %s", url, resp.Status)
			resp.Body.Close()
		} else {
			log.Printf("Keep-alive ping failed: %v", err)
		}
	}
}

func Serve() {
	cnf := config.GetConfig()

	dbCon, err := db.NewConnection(cnf.DatabaseURL)
	if err != nil {
		slog.Error("DB Connect Error", "error", err)
		os.Exit(1)
	}

	err = db.MigrateDB(dbCon, "./migrations")
	if err != nil {
		slog.Error("DB Migrate Error", "error", err)
		os.Exit(1)
	}

	redisDB, err := redis.ConnectRedis(cnf.RedisURL)
	if err != nil {
		slog.Error("Redis Connection Failed", "error", err)
		os.Exit(1)
	}

	storageService := storage.NewStorageService(cnf.Supabase)

	userRepo := repo.NewUserRepo(dbCon)
	userService := user.NewService(userRepo, cnf.JwtSecretKey, storageService)
	userHandler := user_handler.NewHandler(userService, storageService)

	productRepo := repo.NewProductRepo(dbCon)
	productCache := repo.NewProductCache(redisDB)
	productService := product.NewService(productRepo, productCache)
	productHandler := product_handler.NewHandler(productService, storageService)

	cartRepo := repo.NewCartRepo(dbCon)
	orderRepo := repo.NewOrderRepo(dbCon)
	orderService := order.NewService(cartRepo, orderRepo, productService)
	orderHandler := order_handler.NewHandler(orderService)

	orderVerifier := repo.NewOrderVerifier(dbCon)
	reviewRepo := repo.NewReviewRepo(dbCon)
	reviewService := review.NewService(reviewRepo, orderVerifier)
	reviewHandler := review_handler.NewHandler(reviewService)

	chatRepo := repo.NewChatRepo(dbCon)
	chatPubSub := repo.NewChatPubSub(redisDB)
	chatService := chat_pkg.NewService(chatRepo, chatPubSub)
	chatHandler := chat_handler.NewWebSocketHandler(chatService)

	server := rest.NewServer(
		cnf.HttpPort,
		cnf.JwtSecretKey,
		userHandler,
		productHandler,
		orderHandler,
		reviewHandler,
		chatHandler,
	)

	if cnf.BaseURL != "" && cnf.BaseURL != "http://localhost:8080/" {
		go keepAlive(cnf.BaseURL)
	}

	server.Start()
}
