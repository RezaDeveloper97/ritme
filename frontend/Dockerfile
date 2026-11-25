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
ARG NEXT_PUBLIC_API_BASE_URL=https://api.ritme.app/api/v1
ENV NEXT_PUBLIC_API_BASE_URL=$NEXT_PUBLIC_API_BASE_URL \
    NEXT_TELEMETRY_DISABLED=1

# `/[locale]/*` pages are prerendered, and prerendering fetches the UI messages
# from the LIVE API (src/shared/i18n/messages.ts), so the HTML bakes whatever
# that API serves at build time. With identical source this layer would be
# served from cache and keep the previous deploy's strings (T-M2-33, bug B1).
# The deploy scripts pass a fresh BUILD_REV each time; an ARG is part of the
# cache key of every RUN after it, so only `npm run build` (not `npm ci`)
# re-runs. Unset (local builds, today's Laravel prod) = the old caching.
ARG BUILD_REV=
# Build speed on the small build server (2 slow cores, 4 GB):
#  - Lint and type-check are skipped HERE only (next.config.ts reads NEXT_SKIP_BUILD_CHECKS). They are the job of
#    /verify-all before a deploy; repeating them in the image cost ~4 min per build.
#  - webpack's persistent cache survives between builds in a BuildKit cache mount, so only changed modules are
#    recompiled. ONLY `.next/cache/webpack` is mounted: `.next/cache/fetch-cache` must stay empty, or prerendering
#    would reuse the previous deploy's UI messages (the bug BUILD_REV exists for).
ENV NEXT_SKIP_BUILD_CHECKS=1
RUN --mount=type=cache,id=ritme-frontend-next-webpack,target=/app/.next/cache/webpack,sharing=locked \
    echo "frontend build rev: ${BUILD_REV:-none}" && npm run build

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
