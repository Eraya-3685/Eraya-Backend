package main

// @title Eraya Ecommerce API
// @version 1.0
// @description This is a professional ecommerce server for Eraya.
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host https://eraya-backend.onrender.com
// @BasePath /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization

// @tag.name users
// @tag.description User management and authentication

// @tag.name products
// @tag.description Product catalog and details

// @tag.name cart
// @tag.description Shopping cart management

// @tag.name orders
// @tag.description Order placement and history

// @tag.name reviews
// @tag.description Product ratings and comments

// @tag.name chat
// @tag.description Real-time customer support

// @tag.name admin
// @tag.description Administrative operations

import "eraya/cmd"

func main() {
	cmd.Serve()
}
