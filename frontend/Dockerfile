# syntax=docker/dockerfile:1

# ─────────────────────────────────────────────────────────────────────────────
# Ritme (ریتمی) — production image.
# Multi-stage build on top of Next.js `output: 'standalone'` so the final image
# ships only the compiled server + its minimal node_modules, not the source tree
# or dev dependencies.
# ─────────────────────────────────────────────────────────────────────────────

# ---- Base -------------------------------------------------------------------
FROM node:22-alpine AS base
# libc compat for some native deps under Alpine.
RUN apk add --no-cache libc6-compat
WORKDIR /app

# ---- Dependencies -----------------------------------------------------------
# Installed in its own layer so it is cached until the lockfile changes.
FROM base AS deps
COPY package.json package-lock.json ./
RUN npm ci

# ---- Builder ----------------------------------------------------------------
FROM base AS builder
COPY --from=deps /app/node_modules ./node_modules
COPY . .

# NEXT_PUBLIC_* values are inlined into the bundle at build time. They are
# public config only (no secrets / no health data — CLAUDE.md §11). Defaults
# mirror .env.example; override with --build-arg for other environments.
ARG NEXT_PUBLIC_API_BASE_URL=https://ritmeapp.ir/api/v1
ARG NEXT_PUBLIC_OTP_TEST_MODE=true
ENV NEXT_PUBLIC_API_BASE_URL=$NEXT_PUBLIC_API_BASE_URL \
    NEXT_PUBLIC_OTP_TEST_MODE=$NEXT_PUBLIC_OTP_TEST_MODE \
    NEXT_TELEMETRY_DISABLED=1

RUN npm run build

# ---- Runner -----------------------------------------------------------------
FROM base AS runner
ENV NODE_ENV=production \
    NEXT_TELEMETRY_DISABLED=1 \
    PORT=3000 \
    HOSTNAME=0.0.0.0

# Run as an unprivileged user.
RUN addgroup --system --gid 1001 nodejs \
 && adduser --system --uid 1001 nextjs

# Standalone server + the assets it can't inline (static chunks, public/).
COPY --from=builder --chown=nextjs:nodejs /app/.next/standalone ./
COPY --from=builder --chown=nextjs:nodejs /app/.next/static ./.next/static
COPY --from=builder --chown=nextjs:nodejs /app/public ./public

USER nextjs
EXPOSE 3000

CMD ["node", "server.js"]
