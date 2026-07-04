package cmd

import (
	"eraya/aichat"
	chat_pkg "eraya/chat"
	"eraya/config"
	"eraya/infra/bkash"
	"eraya/infra/db"
	"eraya/infra/mail"
	"eraya/infra/redis"
	"eraya/infra/storage"
	"eraya/order"
	"eraya/product"
	"eraya/repo"
	"eraya/rest"
	aichat_handler "eraya/rest/handlers/aichat"
	chat_handler "eraya/rest/handlers/chat"
	order_handler "eraya/rest/handlers/order"
	product_handler "eraya/rest/handlers/product"
	review_handler "eraya/rest/handlers/review"
	settings_handler "eraya/rest/handlers/settings"
	user_handler "eraya/rest/handlers/user"
	wishlist_handler "eraya/rest/handlers/wishlist"
	coupon_handler "eraya/rest/handlers/coupon"
	"eraya/review"
	"eraya/settings"
	"eraya/user"
	"eraya/wishlist"
	"eraya/coupon"
	"log/slog"

	"context"
	"net/http"
	"os"
	"time"
)

func keepAlive(url string) {
	ticker := time.NewTicker(14 * time.Minute)
	for range ticker.C {
		resp, err := http.Get(url)
		if err == nil {
			resp.Body.Close()
		}
	}
}
func startCleanupWorker(svc user.Service) {
	ticker := time.NewTicker(6 * time.Hour) // Run every 6 hours
	for range ticker.C {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
		_ = svc.CleanupUnverifiedUsers(ctx)
		cancel()
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
	mailer := mail.NewMailer(cnf.SMTP, cnf.FrontendURL)

	userRepo := repo.NewUserRepo(dbCon)
	userService := user.NewService(userRepo, cnf.JwtSecretKey, storageService, redisDB, mailer)
	userHandler := user_handler.NewHandler(userService, storageService, cnf.JwtSecretKey)

	productRepo := repo.NewProductRepo(dbCon)
	productCache := repo.NewProductCache(redisDB)
	categoryCache := repo.NewCategoryCache(redisDB)
	productService := product.NewService(productRepo, productCache, categoryCache, storageService)
	productHandler := product_handler.NewHandler(productService, storageService)

	cartRepo := repo.NewCartRepo(dbCon)
	orderRepo := repo.NewOrderRepo(dbCon)
	settingsRepo := repo.NewSettingsRepo(dbCon, redisDB)
	settingsService := settings.NewService(settingsRepo)
	settingsHandler := settings_handler.NewHandler(settingsService, storageService)

	bkashClient := bkash.NewClient(cnf.BKash)

	// Initialize Chat Service first because Order Service depends on it for Stats
	chatRepo := repo.NewChatRepo(dbCon)
	chatPubSub := repo.NewChatPubSub(redisDB)
	chatService := chat_pkg.NewService(chatRepo, chatPubSub)
	chatHandler := chat_handler.NewHandler(chatService)

	wishlistRepo := repo.NewWishlistRepo(dbCon)
	wishlistService := wishlist.NewService(wishlistRepo)
	wishlistHandler := wishlist_handler.NewHandler(wishlistService)

	orderVerifier := repo.NewOrderVerifier(dbCon)
	reviewRepo := repo.NewReviewRepo(dbCon)
	reviewService := review.NewService(reviewRepo, orderVerifier)
	reviewHandler := review_handler.NewHandler(reviewService, storageService)

	couponRepo := repo.NewCouponRepo(dbCon)
	couponService := coupon.NewService(couponRepo)
	couponHandler := coupon_handler.NewHandler(couponService)

	orderService := order.NewService(cartRepo, orderRepo, productService, settingsService, mailer, userService, chatService, couponService)
	orderHandler := order_handler.NewHandler(orderService, bkashClient)

	// AI Chat Service
	aiRepo := repo.NewAIChatRepo(dbCon)
	aiService := aichat.NewService(aiRepo, cnf.AI.GeminiAPIKey, cnf.AI.GroqAPIKey, productService, cnf.FrontendURL)
	aiHandler := aichat_handler.NewHandler(aiService)

	server := rest.NewServer(
		cnf.HttpPort,
		cnf.JwtSecretKey,
		userService,
		userHandler,
		productHandler,
		orderHandler,
		reviewHandler,
		chatHandler,
		wishlistHandler,
		settingsHandler,
		couponHandler,
		aiHandler,
	)

	if cnf.BaseURL != "" && cnf.BaseURL != "http://localhost:8080/" {
		go keepAlive(cnf.BaseURL)
	}

	go startCleanupWorker(userService)

	server.Start()
}
