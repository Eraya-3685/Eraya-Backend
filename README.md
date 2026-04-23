# 🌟 Eraya E-commerce Backend

Eraya is a robust, production-ready e-commerce backend built with **Go (Golang)**, following Clean Architecture and Hexagonal patterns. It features real-time chat, automated order management, and interactive API documentation.

---

## 📖 Live API Documentation

Instead of manual lists, we provide interactive, industry-standard documentation tools:

*   **🚀 ReDoc (Premium Look):** [http://localhost:8080/docs](http://localhost:8080/docs)
    *Best for readability and understanding the API structure.*
*   **🛠️ Swagger UI (Interactive):** [http://localhost:8080/swagger/index.html](http://localhost:8080/swagger/index.html)
    *Best for testing APIs directly from your browser.*

---

## 🛠️ Technology Stack

*   **Language:** Go (Golang)
*   **Framework:** Chi Router (Lightweight & Fast)
*   **Database:** PostgreSQL (with SQLx)
*   **Cache/OTP:** Redis
*   **Auth:** JWT (JSON Web Tokens)
*   **Documentation:** OpenAPI / Swagger / ReDoc
*   **Messaging:** WebSockets (Real-time Chat)

---

## 🚀 Getting Started

### 1. Prerequisites
- Go 1.21+
- PostgreSQL
- Redis

### 2. Environment Setup
Create a `.env` file in the root directory:
```env
PORT=8080
DATABASE_URL=postgres://user:pass@localhost:5432/eraya?sslmode=disable
JWT_SECRET_KEY=your_secret_key
REDIS_HOST=localhost:6379
```

### 3. Run the Project
```bash
# Install dependencies
go mod tidy

# Generate Swagger docs
go run github.com/swaggo/swag/cmd/swag init

# Start the server
go run main.go
```

---

## 🔐 Security Note

All protected endpoints require a Bearer Token:
- **Header:** `Authorization: Bearer <your_jwt_token>`
- **WebSocket:** Use query param `?token=<your_jwt_token>`

---