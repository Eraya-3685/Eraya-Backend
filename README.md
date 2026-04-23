# 🌟 Eraya E-commerce Backend

Eraya is a robust, production-ready e-commerce backend built with **Go (Golang)**, following Clean Architecture and Hexagonal patterns. It features real-time chat, automated order management, and interactive API documentation.

---

## 📖 Live API Documentation

We provide interactive, industry-standard documentation tools that stay in sync with the code:

*   **🚀 ReDoc (Premium Look):** `YOUR_SERVER_URL/docs`
    *Best for readability and high-level structure.*
*   **🛠️ Swagger UI (Interactive):** `YOUR_SERVER_URL/swagger/index.html`
    *Best for testing APIs directly with live requests.*

> **Note:** Replace `YOUR_SERVER_URL` with `http://localhost:8080` for local development or your production domain (e.g., `https://eraya.onrender.com`).

---

## 🛠️ Architecture & Technology Stack

### Core Principles
- **Hexagonal Architecture:** Decoupled business logic from external drivers (DB, Web).
- **Clean Dependency Injection:** Functions only receive the specific configurations they need.
- **RESTful Design:** Standard HTTP methods and status codes.

### Tech Stack
- **Language:** Go (Golang) 1.21+
- **Database:** PostgreSQL (with SQLx for clean queries)
- **Cache/PubSub:** Redis (Supports TLS for Cloud providers like Upstash)
- **Auth:** JWT (JSON Web Tokens) with Role-Based Access Control (RBAC)
- **Messaging:** WebSockets for real-time chat support.

---

## 🚀 Installation & Setup

### 1. Prerequisites
- Go 1.21 or higher
- PostgreSQL
- Redis (Local or Cloud like Upstash)

### 2. Environment Variables
Create a `.env` file in the root:
```env
PORT=8080
DATABASE_URL="postgresql://user:password@host:port/dbname?sslmode=disable"
REDIS_URL="rediss://default:password@host:port"
JWT_SECRET_KEY="your-super-secret-key"
```

### 3. Execution Commands
```bash
# Install dependencies
go mod tidy

# Update Swagger Docs (whenever annotations change)
go run github.com/swaggo/swag/cmd/swag init

# Run the server
go run main.go
```

---

## 🔐 Security & Integration

- **Authorization:** All protected routes require `Authorization: Bearer <token>`
- **Chat Auth:** For WebSockets, pass the token as a query parameter: `?token=<token>`
- **Database Migrations:** Automatically runs migrations from the `/migrations` folder on startup.

---
*Built with ❤️ for Eraya Team*