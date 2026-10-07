# syntax=docker/dockerfile:1

# ─────────────────────────────────────────────────────────────────────────────
# Ritme instructor-web — production image (Next.js `output: 'standalone'`).
# Internal only: nginx on the instructor host proxies `/` here and `/api/` to
# backend-go, so the browser sees one origin (deploy/vhost-instructor.inc,
# docs/night-bloom/instructor-web-deploy.md).
# ─────────────────────────────────────────────────────────────────────────────

FROM node:22-alpine AS base
RUN apk add --no-cache libc6-compat
WORKDIR /app

FROM base AS deps
COPY package.json package-lock.json ./
RUN npm ci

FROM base AS builder
COPY --from=deps /app/node_modules ./node_modules
COPY . .
# Inlined into the browser bundle at build time; public config only. Relative
# paths = same origin as the instructor host (stage and prod).
ARG NEXT_PUBLIC_API_BASE_URL=/api/v1
ARG NEXT_PUBLIC_INSTRUCTOR_API_BASE_URL=/api/instructor/v1
ENV NEXT_PUBLIC_API_BASE_URL=$NEXT_PUBLIC_API_BASE_URL \
    NEXT_PUBLIC_INSTRUCTOR_API_BASE_URL=$NEXT_PUBLIC_INSTRUCTOR_API_BASE_URL \
    NEXT_TELEMETRY_DISABLED=1
RUN npm run build

FROM base AS runner
ENV NODE_ENV=production \
    NEXT_TELEMETRY_DISABLED=1 \
    PORT=3000 \
    HOSTNAME=0.0.0.0
RUN addgroup --system --gid 1001 nodejs \
 && adduser --system --uid 1001 nextjs
COPY --from=builder --chown=nextjs:nodejs /app/.next/standalone ./
COPY --from=builder --chown=nextjs:nodejs /app/.next/static ./.next/static
COPY --from=builder --chown=nextjs:nodejs /app/public ./public
USER nextjs
EXPOSE 3000
HEALTHCHECK --interval=15s --timeout=3s --start-period=10s --retries=3 \
  CMD wget -qO /dev/null "http://127.0.0.1:3000/login" || exit 1
CMD ["node", "server.js"]
