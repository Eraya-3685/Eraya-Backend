<p align="center">
  <img src="https://img.shields.io/badge/Go-1.26-00ADD8?style=for-the-badge&logo=go&logoColor=white" alt="Go" />
  <img src="https://img.shields.io/badge/PostgreSQL-16-4169E1?style=for-the-badge&logo=postgresql&logoColor=white" alt="PostgreSQL" />
  <img src="https://img.shields.io/badge/Redis-Upstash-DC382D?style=for-the-badge&logo=redis&logoColor=white" alt="Redis" />
  <img src="https://img.shields.io/badge/Supabase-Storage-3ECF8E?style=for-the-badge&logo=supabase&logoColor=white" alt="Supabase" />
  <img src="https://img.shields.io/badge/Swagger-API_Docs-85EA2D?style=for-the-badge&logo=swagger&logoColor=black" alt="Swagger" />
</p>

<h1 align="center">🛍️ Eraya — Backend API</h1>

<p align="center">
  <strong>A production-grade RESTful ecommerce backend built with Go, powering the Eraya fashion marketplace.</strong>
</p>

<p align="center">
  <a href="https://eraya-backend.onrender.com/docs">📖 API Docs (ReDoc)</a> •
  <a href="https://eraya-backend.onrender.com/swagger/">📋 Swagger UI</a> •
  <a href="https://eraya-backend.onrender.com/health">❤️ Health Check</a>
</p>

---

## 🏗️ Architecture

Eraya's backend follows a **clean architecture** (Domain-Driven Design) pattern with clear separation of concerns:

```
Backend/
├── main.go              # Application entry point (Swagger annotations)
├── cmd/                 # Server bootstrap & dependency injection
├── config/              # Environment configuration loader
├── domain/              # Domain models & repository interfaces
├── repo/                # PostgreSQL repository implementations + Redis caching
├── rest/                # HTTP layer
│   ├── handlers/        # Route handlers by domain (user, product, order, ...)
│   ├── middlewares/     # Auth, CORS, rate limiting, permissions
│   └── server.go        # Chi router setup & route registration
├── infra/               # External service integrations
│   ├── bkash/           # bKash payment gateway client
│   ├── db/              # PostgreSQL connection & migrations
│   ├── mail/            # SMTP email service (OTP, order confirmations)
│   ├── redis/           # Redis/Upstash connection
│   └── storage/         # Supabase Storage file uploads
├── migrations/          # SQL migration files (17 incremental migrations)
├── docs/                # Auto-generated Swagger documentation
├── user/                # User business logic service
├── product/             # Product business logic service
├── order/               # Order & cart business logic service
├── review/              # Review business logic service
├── chat/                # Real-time WebSocket chat service
├── aichat/              # AI shopping assistant (Gemini + Groq)
├── wishlist/            # Wishlist service
├── coupon/              # Coupon/discount service
├── settings/            # Store configuration service
└── util/                # JWT, hashing, file handling, response helpers
```

---

## ⚡ Tech Stack

| Layer            | Technology                                                                                        |
| ---------------- | ------------------------------------------------------------------------------------------------- |
| **Language**     | Go 1.26                                                                                          |
| **Router**       | [Chi v5](https://github.com/go-chi/chi) — lightweight, composable HTTP router                    |
| **Database**     | PostgreSQL (via Supabase) with [sqlx](https://github.com/jmoiron/sqlx)                           |
| **Cache**        | Redis (Upstash) — product & category caching, chat pub/sub                                       |
| **Auth**         | JWT (HS256) via [golang-jwt/jwt](https://github.com/golang-jwt/jwt) + bcrypt password hashing   |
| **WebSocket**    | [Gorilla WebSocket](https://github.com/gorilla/websocket) — real-time chat & order notifications |
| **File Storage** | Supabase Storage — product images, avatars, review photos, logos                                 |
| **Payments**     | bKash Tokenized Checkout API (sandbox + mock support)                                            |
| **Email**        | SMTP (Gmail) — OTP verification, order confirmations, password resets                            |
| **AI**           | Google Gemini 2.0 Flash (primary) + Groq Llama 3.3 70B (fallback)                                |
| **Migrations**   | [sql-migrate](https://github.com/rubenv/sql-migrate) — versioned schema migrations              |
| **API Docs**     | [Swag](https://github.com/swaggo/swag) + Swagger UI + ReDoc                                     |
| **Deployment**   | Render (with auto keep-alive ping every 14 minutes)                                              |

---

## 🔌 API Endpoints

All routes are prefixed with `/api/v1`. Authentication uses `Bearer <JWT>` in the `Authorization` header.

### 👤 Users — `/api/v1/users`

| Method   | Endpoint                 | Auth     | Description                        |
| -------- | ------------------------ | -------- | ---------------------------------- |
| `POST`   | `/signup`                | Public   | Register with email & password     |
| `POST`   | `/verify-signup`         | Public   | Verify OTP to activate account     |
| `POST`   | `/resend-activation`     | Public   | Resend activation OTP              |
| `POST`   | `/login`                 | Public   | Login with credentials             |
| `POST`   | `/social-login`          | Public   | Google/GitHub OAuth via Supabase   |
| `POST`   | `/forgot-password`       | Public   | Send password reset OTP            |
| `POST`   | `/reset-password`        | Public   | Reset password with OTP            |
| `GET`    | `/profile`               | User     | Get current user profile           |
| `PATCH`  | `/profile`               | User     | Update profile (name, phone, etc.) |
| `PATCH`  | `/avatar`                | User     | Upload profile avatar              |
| `POST`   | `/otp/request`           | User     | Request OTP for secure operations  |
| `POST`   | `/otp/verify`            | User     | Verify OTP                         |
| `PATCH`  | `/secure-update`         | User     | Update email/phone with OTP        |
| `POST`   | `/change-password`       | User     | Change password                    |
| `GET`    | `/`                      | Mod/Admin| List all users (paginated)         |
| `GET`    | `/{id}`                  | Mod/Admin| Get user by ID                     |
| `PATCH`  | `/{id}/role`             | Admin    | Update user role                   |
| `POST`   | `/bulk-role`             | Admin    | Bulk update user roles             |
| `DELETE` | `/{id}`                  | Admin    | Delete user                        |

### 📦 Products — `/api/v1/products`

| Method   | Endpoint             | Auth     | Description                               |
| -------- | -------------------- | -------- | ----------------------------------------- |
| `GET`    | `/`                  | Public   | List products (search, filter, sort, page) |
| `GET`    | `/{slug}`            | Public   | Get product by slug                        |
| `POST`   | `/`                  | Mod/Admin| Create product (multipart with images)     |
| `PUT`    | `/{id}`              | Mod/Admin| Update product                             |
| `DELETE` | `/{id}`              | Mod/Admin| Delete product                             |
| `POST`   | `/bulk-delete`       | Mod/Admin| Bulk delete products                       |

### 🏷️ Categories — `/api/v1/categories`

| Method   | Endpoint             | Auth     | Description                |
| -------- | -------------------- | -------- | -------------------------- |
| `GET`    | `/`                  | Public   | List all categories        |
| `POST`   | `/`                  | Mod/Admin| Create category            |
| `PUT`    | `/{id}`              | Mod/Admin| Update category            |
| `DELETE` | `/{id}`              | Mod/Admin| Delete category            |
| `POST`   | `/bulk-delete`       | Mod/Admin| Bulk delete categories     |

### 🛒 Cart — `/api/v1/cart`

| Method | Endpoint | Auth | Description                           |
| ------ | -------- | ---- | ------------------------------------- |
| `POST` | `/`      | User | Add item to cart (color, size, qty)   |
| `GET`  | `/`      | User | Get cart items                        |

### 🧾 Orders — `/api/v1/orders`

| Method   | Endpoint                       | Auth     | Description                         |
| -------- | ------------------------------ | -------- | ----------------------------------- |
| `POST`   | `/checkout`                    | User     | Place order (COD or bKash)          |
| `POST`   | `/bkash/init`                  | User     | Initialize bKash payment            |
| `GET`    | `/bkash/callback`              | Public   | bKash payment callback              |
| `GET`    | `/`                            | User     | List my orders                      |
| `GET`    | `/{id}`                        | User     | Get order details                   |

### 🔧 Admin Orders — `/api/v1/admin/orders`

| Method   | Endpoint                       | Auth     | Description                         |
| -------- | ------------------------------ | -------- | ----------------------------------- |
| `GET`    | `/ws`                          | Mod/Admin| WebSocket for real-time order updates|
| `GET`    | `/`                            | Mod/Admin| List all orders (search, filter)    |
| `GET`    | `/stats`                       | Mod/Admin| Dashboard statistics                |
| `POST`   | `/{id}/confirm`                | Mod/Admin| Confirm order                       |
| `PUT`    | `/{id}/status`                 | Mod/Admin| Update order status                 |
| `POST`   | `/request-delete-otp`          | Mod/Admin| Request OTP for order deletion      |
| `DELETE` | `/{id}`                        | Mod/Admin| Delete order (OTP-protected)        |

### ⭐ Reviews — `/api/v1/reviews`

| Method   | Endpoint                    | Auth     | Description                    |
| -------- | --------------------------- | -------- | ------------------------------ |
| `GET`    | `/{productId}`              | Public   | Get product reviews            |
| `POST`   | `/`                         | User     | Create review (verified buyer) |
| `POST`   | `/upload`                   | User     | Upload review image            |
| `DELETE` | `/{id}`                     | Admin    | Delete review                  |
| `GET`    | `/admin/reviews/`           | Admin    | List all reviews               |
| `POST`   | `/admin/reviews/{id}/approve`| Admin   | Approve/reject review          |

### 💬 Chat — `/api/v1/chat`

| Method   | Endpoint                       | Auth | Description                         |
| -------- | ------------------------------ | ---- | ----------------------------------- |
| `GET`    | `/ws`                          | User | WebSocket connection for real-time chat |
| `GET`    | `/conversations`               | User | List conversations                  |
| `GET`    | `/conversation/{withID}`       | User | Get/create conversation with user   |
| `DELETE` | `/conversation/{id}`           | User | Delete conversation                 |
| `POST`   | `/conversation/{id}/read`      | User | Mark messages as read               |
| `POST`   | `/messages/bulk-delete`        | User | Bulk delete messages                |
| `GET`    | `/users/search`                | User | Search users for new conversations  |
| `GET`    | `/unread-count`                | User | Get unread message count            |

### 🤖 AI Assistant — `/api/v1/ai`

| Method | Endpoint | Auth     | Description                            |
| ------ | -------- | -------- | -------------------------------------- |
| `POST` | `/chat`  | Optional | Chat with AI shopping assistant        |
| `GET`  | `/chat`  | Optional | Get conversation history (logged in)   |

### ❤️ Wishlist — `/api/v1/wishlist`

| Method   | Endpoint            | Auth | Description                 |
| -------- | ------------------- | ---- | --------------------------- |
| `GET`    | `/`                 | User | Get wishlist                |
| `POST`   | `/{product_id}`     | User | Add to wishlist             |
| `DELETE` | `/{product_id}`     | User | Remove from wishlist        |
| `DELETE` | `/`                 | User | Clear entire wishlist       |

### 🎟️ Coupons — `/api/v1/coupons`

| Method   | Endpoint             | Auth  | Description                      |
| -------- | -------------------- | ----- | -------------------------------- |
| `POST`   | `/apply`             | User  | Apply coupon to cart             |
| `POST`   | `/admin/coupons/`    | Admin | Create coupon                    |
| `GET`    | `/admin/coupons/`    | Admin | List all coupons                 |
| `DELETE` | `/admin/coupons/{id}`| Admin | Delete coupon                    |

### ⚙️ Settings — `/api/v1/settings`

| Method | Endpoint | Auth     | Description                      |
| ------ | -------- | -------- | -------------------------------- |
| `GET`  | `/`      | Public   | Get store settings               |
| `PUT`  | `/`      | Mod/Admin| Update store settings            |
| `POST` | `/logo`  | Mod/Admin| Upload store logo                |

### 📤 File Upload — `/api/v1/upload`

| Method | Endpoint  | Auth     | Description              |
| ------ | --------- | -------- | ------------------------ |
| `POST` | `/upload` | Mod/Admin| Upload file to Supabase  |

---

## 🔐 Security

| Feature                     | Implementation                                                  |
| --------------------------- | --------------------------------------------------------------- |
| **Authentication**          | JWT Bearer tokens (HS256) with DB-verified active user check    |
| **Password Security**       | bcrypt hashing with salt                                        |
| **OTP Verification**        | Email-based OTP for signup, password reset, secure updates      |
| **Role-Based Access (RBAC)**| 3-tier: `buyer` → `moderator` → `admin`                        |
| **Granular Permissions**    | Moderators have configurable permissions per resource           |
| **Rate Limiting**           | IP-based: 5 req/sec, burst 10 (global) + 20 req/min (AI chat) |
| **Security Headers**        | X-Frame-Options, X-Content-Type-Options, X-XSS-Protection, HSTS|
| **CORS**                    | Configurable allowed origins                                    |
| **Input Validation**        | Request body size limits, message length caps                   |

---

## 🧠 AI Shopping Assistant

The AI assistant is a standout feature that transforms product discovery:

- **Primary Model**: Google Gemini 2.0 Flash — fast, context-aware responses
- **Fallback Model**: Groq Llama 3.3 70B — automatic failover for high availability
- **Product-Aware**: Dynamically queries the product catalog based on user intent
- **Bilingual**: Supports both English and Bangla (with Bangla price parsing: ৳, tk, taka)
- **Smart Search**: Extracts keywords and price constraints from natural language
- **Conversation History**: Persisted per user, supports up to 50 messages
- **Rate Limited**: 20 requests/minute per user to prevent abuse

---

## 💳 Payment Integration

### bKash Tokenized Checkout

- Full integration with bKash's tokenized payment API
- **Create Payment** → **Redirect to bKash** → **Callback Verification** → **Execute Payment**
- Sandbox mode with mock payment support for development
- Automatic stock adjustment on successful payment

### Cash on Delivery (COD)

- Direct checkout with shipping address
- Admin confirmation workflow

---

## 📊 Database Schema

The database has **17 incremental migrations** managing:

| Table                  | Description                                          |
| ---------------------- | ---------------------------------------------------- |
| `users`                | User accounts with roles, social IDs, OTP support    |
| `user_permissions`     | Granular moderator permission assignments            |
| `products`             | Product catalog with variations (colors, sizes)      |
| `product_images`       | Multiple images per product with primary flag        |
| `product_categories`   | Many-to-many product ↔ category relationship         |
| `categories`           | Product categories with images                       |
| `cart_items`           | Per-user shopping cart with color/size selection      |
| `orders`               | Order records with full lifecycle tracking           |
| `order_items`          | Individual items in each order                       |
| `reviews`              | Product reviews with ratings, images, verification   |
| `conversations`        | Chat conversations between buyers and admins         |
| `messages`             | Chat messages with attachments & reply threading     |
| `wishlists`            | User product wishlists                               |
| `store_settings`       | Global store configuration (fees, contact, logo)     |
| `coupons`              | Discount coupons (percentage or flat, expiry, min)   |
| `ai_chat_history`      | AI assistant conversation persistence                |
| `product_variations`   | Color/size stock tracking per variant                |

---

## 🚀 Getting Started

### Prerequisites

- Go 1.26+
- PostgreSQL 16+
- Redis (or Upstash)
- Supabase project (for storage)

### Setup

```bash
# Clone the repository
git clone <repo-url>
cd Backend

# Copy environment template and fill in values
cp .env.example .env

# Install dependencies
go mod download

# Run database migrations (automatic on startup)
# Migrations run automatically when the server starts

# Start the development server
go run main.go
```

### Environment Variables

```env
# Server
BASE_URL=http://localhost:8080/
FRONTEND_URL=http://localhost:5173/

# Database
DATABASE_URL=postgresql://user:pass@host:port/dbname

# Redis
REDIS_URL=rediss://default:token@host:port

# Authentication
JWT_SECRET_KEY=your-jwt-secret

# Supabase Storage
SUPABASE_URL=https://your-project.supabase.co
SUPABASE_SERVICE_KEY=your-service-key
SUPABASE_BUCKET=eraya

# Email (SMTP)
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=your-email@gmail.com
SMTP_PASS=your-app-password

# bKash Payment
BKASH_USERNAME=sandbox-user
BKASH_PASSWORD=sandbox-pass
BKASH_APP_KEY=your-app-key
BKASH_APP_SECRET=your-app-secret
BKASH_BASE_URL=https://tokenized.sandbox.bka.sh/v1.2.0-beta

# AI
GEMINI_API_KEY=your-gemini-key
GROQ_API_KEY=your-groq-key
```

### Generate Swagger Docs

```bash
# Install swag CLI
go install github.com/swaggo/swag/cmd/swag@latest

# Generate docs
swag init
```

---

## 📡 Real-Time Features

### WebSocket Endpoints

| Endpoint                        | Purpose                                   |
| ------------------------------- | ----------------------------------------- |
| `/api/v1/chat/ws`               | Real-time buyer ↔ admin messaging         |
| `/api/v1/admin/orders/ws`       | Live order notifications for admin panel  |

### Chat Features

- **Redis Pub/Sub**: Cross-instance message broadcasting
- **Presence Detection**: Online/offline status tracking
- **Typing Indicators**: Real-time typing state
- **Message Threading**: Reply-to functionality
- **File Attachments**: Image sharing in conversations
- **Read Receipts**: Mark messages as read

---

## 🏭 Background Workers

| Worker                    | Schedule       | Description                         |
| ------------------------- | -------------- | ----------------------------------- |
| **Keep-Alive Ping**       | Every 14 min   | Prevents Render free-tier sleep     |
| **Unverified User Cleanup**| Every 6 hours | Removes unverified accounts         |

---