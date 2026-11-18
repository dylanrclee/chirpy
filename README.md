Chirpy
- Chripy is a lightweight RESTful microblogging API built in Go. It allows users to register, authenticate users with JWTs, post messages, and manage user subscriptions with external webhook integrations.


Features
- User Authentication
- Message(Chirp) Management: Full CRUD operations for messages (called chirps) with message filtering options.
- Webhook Integration: Upgrade users to "Chirpy Red" via Stripe_style payment webhooks.
- PostgreSQL Database: Uses SQL queries to interact with database.


Built With
- Go (Golang): Core API server and routing
- PostgreSQL: Database
- sqlc: Generated Go code from SQL
- Goose: Database migrations


Prerequisites
- Go 1.27 or higher
- PostgreSQL running locally
- Goose (for migrations)


Getting Started/Installation
1. Clone the repository:
    git clone https://github.com/dylanrclee/chirpy.git
    cd chirpy
2. Configure environemt variables:
    Set up .env file with DB_URL, PLATFORM, SECRET, and POLKA_KEY set.
    DB_URL needs to be set to current PostgreSQL instance.
3. Run database migrations:
    cd sql/schema
    goose postgres "postgres://postgres:password@localhost:5432/chirpy" up
    cd ../..
4. Run the server:
    go build -o out && ./out


API Documentation
- GET /api/healthz
- GET /admin/metrics
- GET /api/chirps
- GET /api/chirps/{chirpID}
- POST /admin/reset
- POST /api/validate_chirp
- POST /api/users
- POST /api/chirps
- POST /api/login
- POST /api/refresh
- POST /api/revoke
- POST /api/polka/webhooks
- PUT /api/users
- DELETE /api/chirps/{chirpID}