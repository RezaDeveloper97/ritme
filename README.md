# Ritme (ریتمی)

Monorepo for the Ritme women's-health application.

```
ritme/
├── backend/    # Laravel 12 API (PHP 8.3, Passport OTP auth, MySQL, Redis)
├── frontend/   # Next.js app (App Router, next-intl fa/en, FSD architecture)
└── docker-compose.yml
```

## Run with Docker

```bash
cp .env.example .env   # then set APP_KEY and the DB passwords
docker compose up -d --build
```

| Service  | URL / port                                   |
| -------- | -------------------------------------------- |
| frontend | http://localhost:3000                        |
| backend  | http://localhost:8080 (API under `/api/v1`)  |
| swagger  | http://localhost:8080/api/documentation      |
| mysql    | localhost:3307 (inside the network: `mysql`) |
| redis    | internal only (service name: `redis`)        |

On first boot the backend container runs migrations, generates Passport keys,
and creates the personal-access client automatically. A separate `queue`
container runs `php artisan queue:work` against Redis.

### Data persistence

MySQL data, Redis data (AOF), and Laravel's `storage/` (incl. Passport keys)
live in named volumes — rebuilding images (`docker compose up -d --build`)
does **not** wipe them. Only `docker compose down -v` deletes data.

### Frontend build args

`NEXT_PUBLIC_*` values are inlined at build time. To point the frontend at a
different API, change `NEXT_PUBLIC_API_BASE_URL` in `.env` and rebuild:

```bash
docker compose build frontend && docker compose up -d frontend
```

`NEXT_PUBLIC_OTP_TEST_MODE=true` makes login accept `1111` as the OTP without
sending real SMS — set it to `false` (and fill the SMS provider keys) for
production.
