# gRPC + Casdoor POC

Authentication with Casdoor, gRPC and PostgreSQL.

---

## Architecture

```
Vue  →  gRPC (Login/Register)  →  gRPC Server  ↔  Casdoor (hidden)
                                     ↓
                               PostgreSQL (your DB)
```

| Component   | Port   | Role                                          |
| ----------- | ------ | --------------------------------------------- |
| Casdoor     | :8000  | Identity Provider — manages users, issues JWT |
| gRPC Server | :50051 | Main API — Login, Register, Ping              |
| PostgreSQL  | :5432  | Your database — stores users                  |

---

## Prerequisites

| Tool           | Min. Version | Usage                    |
| -------------- | ------------ | ------------------------ |
| Go             | 1.21+        | Main language            |
| Docker Desktop | 4.x          | Casdoor and PostgreSQL   |
| buf CLI        | 1.x          | Protobuf code generation |
| Postman        | Latest       | Testing gRPC endpoints   |

---

## Step 1 — Docker Setup (Casdoor + PostgreSQL)

Create a `docker-compose.yml` at the root of the project:

```bash
docker run -d --name casdoor -p 8000:8000 casbin/casdoor-all-in-one
```

Start both services:

```bash
docker-compose up -d
```

Verify everything is running:

```bash
docker ps
```

> Casdoor available at `http://localhost:8000` — PostgreSQL at `localhost:5432`

---

## Step 2 — Casdoor Configuration

Open `http://localhost:8000` in your browser. Default credentials: `admin` / `123`.

### 2.1 — Create the organization

1. Go to **Organizations → Add**
2. Set the **Name** to `admin` (or your project name)
3. Save

### 2.2 — Create the application

1. Go to **Applications → Add**
2. Fill in the following fields:

| Field         | Value                             |
| ------------- | --------------------------------- |
| Name          | grpc-app                          |
| Organization  | admin                             |
| Redirect URIs | http://localhost:9999/callback    |
| Grant types   | Authorization Code + **Password** |

3. Save and **copy the Client ID and Client Secret** that are generated.

> Make sure **Password** is checked in the Grant types. Without it, the gRPC Login will return `unsupported_grant_type`.

---

## Step 3 — Create the PostgreSQL table

Connect to PostgreSQL:

```bash
docker exec -it <postgres_container_name> psql -U postgres -d grpcpoc
```

Then run:

```sql
CREATE TABLE users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  casdoor_id VARCHAR(255) UNIQUE NOT NULL,
  email VARCHAR(255) UNIQUE NOT NULL,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
```

Verify the table was created:

```sql
\dt
```

---

## Step 4 — Environment Configuration

Create a `.env` file at the root of the project:

```env
CLIENT_ID="<your_casdoor_client_id>"
CLIENT_SECRET="<your_casdoor_client_secret>"
CASDOOR_URL="http://localhost:8000"
SERVER_URL="localhost:50051"
SERVER_PORT=":50051"
CALLBACK_URL="http://localhost:9999/callback"
AUTH_URL="http://localhost:8000/login/oauth/authorize"
TOKEN_URL="http://localhost:8000/api/login/oauth/access_token"
DATABASE_URL="postgres://postgres:postgres@localhost:5432/grpcpoc?sslmode=disable"
ORGANIZATION="admin"
APP_NAME="grpc-app"
```

---

## Step 5 — Protobuf Code Generation

The proto file defines your API contract. After any modification, regenerate the code:

```bash
buf generate
```

> Files in `gen/go/proto/auth/v1/` are auto-generated — never edit them manually.

---

## Step 6 — Start the Server

In this order:

**1. Start Docker (Casdoor + PostgreSQL)**

```bash
docker-compose up -d
```

**2. Start the gRPC server**

```bash
go run .
```

> Expected output: `gRPC server running on :50051`

---

## Step 7 — Testing with Postman

### Postman setup

1. Click **New → gRPC Request**
2. URL: `localhost:50051`
3. Service definition → **Server reflection**
4. Postman will automatically discover all available methods

### Test Login

Method: `auth.v1.SecureService/Login`

```json
{
  "email": "your_casdoor_username",
  "password": "your_password"
}
```

> Expected response:
>
> ```json
> { "access_token": "eyJ...", "user_id": "uuid...", "email": "..." }
> ```

### Test Register

Method: `auth.v1.SecureService/Register`

```json
{
  "username": "newuser",
  "email": "newuser@example.com",
  "password": "password123"
}
```

### Test Ping (protected route)

Method: `auth.v1.SecureService/Ping`

In the **Metadata** tab, add:

- Key: `authorization`
- Value: `Bearer <access_token_from_login>`

```json
{
  "message": "Hello"
}
```

> Expected response: `{ "message": "Pong: Hello" }`

---

## Project Structure

```
grpc-casdoor-poc/
├── main.go          Entry point — assembles all dependencies
├── internal/
│   ├── auth/interceptor.go     gRPC middleware — verifies JWT tokens
│   ├── casdoor/client.go       HTTP client for the Casdoor API
│   └── store/user.go           PostgreSQL access — upserts users
├── gen/go/proto/auth/v1/       Code generated by buf (do not edit)
│   ├── auth.pb.go
│   └── auth_grpc.pb.go
├── proto/auth/v1/auth.proto    API contract
├── config/config.go            .env loading
├── docker-compose.yml          Casdoor + PostgreSQL
├── .env                        Environment variables (do not commit)
└── go.mod
```

---
