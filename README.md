# Sonet

**Sonet** is a blazing fast, lightweight, and modular API microservice for **Posts, Comments, and Reactions**, built with **Go (Fiber)**. It supports multiple databases (PostgreSQL, SQLite, Firestore, Supabase) via a clean adapter system. Sonet is app-agnostic — just pass a `user_id` and it works seamlessly with any auth system.

> Use Sonet as the drop-in content engine for your social layer — scalable, fast, and extendable.

---

## 🔥 Features

- **Posts, Comments, Reactions (multi-reaction)**: Flexible interaction model like Discord or Slack.
- **User-Agnostic**: Pass `user_id` only — you own auth and profiles.
- **Hook System**: Subscribe to actions (post created, reaction added) for notifications or analytics.
- **Adapter-Based DB Support**: PostgreSQL, SQLite, Firestore, Supabase (others pluggable).
- **High Performance**: Fiber + Go for ultra-low latency APIs.
- **Microservice-Friendly**: Stateless, lightweight, deploy anywhere.
- **Rate Limiting & Anti-Spam (Planned)**: Control abuse.
- **OpenAPI Spec (Planned)**: Auto SDK support and docs.
- **Future Plugins**: Reactions+, user mentions, moderation flags, subscriptions.

---

## 🧰 Tech Stack

| Layer        | Tool               | Description |
|--------------|--------------------|-------------|
| Language     | Go                 | Minimal, compiled, fast backend language |
| Framework    | [Fiber](https://gofiber.io) | Web framework inspired by Express.js, optimized for performance |
| ORM / DB     | GORM (or raw SQL)  | Used for PostgreSQL/SQLite adapters (abstracted) |
| Database     | PostgreSQL, SQLite, Firestore, Supabase | Adapter support for multiple backends |
| Config       | Viper              | Env and config management |
| API Docs     | Swagger (planned)  | OpenAPI spec for client SDK generation |
| Container    | Docker             | For portable deployment |
| Hooks        | Internal & Webhook | Trigger functions or URLs when actions occur |
| Rate Limiting| Fiber Middleware   | Basic anti-abuse protection (planned) |

---

## ✨ Use Cases

- Add social features to an app (posts, comments, likes)
- Build a lightweight discussion or microblogging platform
- Create a comment system for blogs or documentation
- Power interactions in multiplayer or community games
- Use as a shared microservice in a large system

---

## 🧩 Why Sonet vs. Others?

| Feature                  | Sonet        | Artalk | Cusdis | Schnack | SupaComments |
|--------------------------|--------------|--------|--------|---------|--------------|
| Comments + Posts + Likes | ✅ Yes       | ❌     | ❌     | ❌      | ❌           |
| DB-Agnostic               | ✅ Yes       | ❌     | ❌     | ❌      | ❌           |
| Microservice Ready        | ✅ Yes       | ❌     | ✅     | ✅      | ✅           |
| Hooks/Webhooks            | ✅ Yes       | ❌     | ❌     | ❌      | ❌           |
| Bring Your Own User Auth | ✅ Yes       | ❌     | ✅     | ✅      | ✅           |
| Written in Go (Fast)     | ✅ Yes       | ✅     | ❌     | ❌      | ❌           |

---

## 🚀 Quick Start

```bash
git clone https://github.com/ibrahimyu/sonet
cd sonet
cp .env.example .env
# Edit .env file to configure your database
go run cmd/api/main.go
```

### Using Docker

```bash
# Build and run with Docker
docker build -t sonet .
docker run -p 8080:8080 --env-file .env sonet

# Or with Docker Compose (includes PostgreSQL)
docker-compose up -d
```

### Using Make

```bash
# Setup .env file
make setup

# Build the application
make build

# Run the application
make run

# Run with docker-compose
make docker-compose
```

## 📖 API Documentation

See [API.md](./API.md) for detailed API documentation.
