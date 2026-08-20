# syntax=docker/dockerfile:1.7
FROM node:26-alpine AS deps
WORKDIR /app
COPY package.json ./
RUN npm install --no-audit --no-fund

FROM node:26-alpine AS build
WORKDIR /app
ENV NEXT_TELEMETRY_DISABLED=1
ARG NEXT_PUBLIC_FIREBASE_PROJECT_ID=demo-rentstage
ARG NEXT_PUBLIC_FIREBASE_API_KEY=rentstage-local-api-key
ARG NEXT_PUBLIC_FIREBASE_AUTH_DOMAIN=demo-rentstage.firebaseapp.com
ARG NEXT_PUBLIC_FIREBASE_AUTH_EMULATOR_PORT=9099
ARG NEXT_PUBLIC_USE_AUTH_EMULATOR=true
ARG NEXT_PUBLIC_SUPPORT_EMAIL=support@rentstage.local
ARG RENTSTAGE_VERSION=dev
ENV NEXT_PUBLIC_FIREBASE_PROJECT_ID=${NEXT_PUBLIC_FIREBASE_PROJECT_ID}
ENV NEXT_PUBLIC_FIREBASE_API_KEY=${NEXT_PUBLIC_FIREBASE_API_KEY}
ENV NEXT_PUBLIC_FIREBASE_AUTH_DOMAIN=${NEXT_PUBLIC_FIREBASE_AUTH_DOMAIN}
ENV NEXT_PUBLIC_FIREBASE_AUTH_EMULATOR_PORT=${NEXT_PUBLIC_FIREBASE_AUTH_EMULATOR_PORT}
ENV NEXT_PUBLIC_USE_AUTH_EMULATOR=${NEXT_PUBLIC_USE_AUTH_EMULATOR}
ENV NEXT_PUBLIC_SUPPORT_EMAIL=${NEXT_PUBLIC_SUPPORT_EMAIL}
ENV RENTSTAGE_VERSION=${RENTSTAGE_VERSION}
COPY --from=deps /app/node_modules ./node_modules
COPY . .
RUN npm run test:ci

FROM node:26-alpine AS runner
WORKDIR /app
ARG VCS_REF=unknown
ARG BUILD_DATE=unknown
ARG RENTSTAGE_VERSION=dev
LABEL org.opencontainers.image.title="RentStage Web" \
      org.opencontainers.image.version="${RENTSTAGE_VERSION}" \
      org.opencontainers.image.revision="${VCS_REF}" \
      org.opencontainers.image.created="${BUILD_DATE}" \
      org.opencontainers.image.source="https://github.com/rentstage/rentstage"
ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1
ENV PORT=3000
ENV HOSTNAME=0.0.0.0
ENV RENTSTAGE_VERSION=${RENTSTAGE_VERSION}
RUN addgroup --system --gid 1001 nodejs && adduser --system --uid 1001 nextjs

# The standalone server requires node, but not npm or npx.
RUN rm -rf /usr/local/lib/node_modules/npm \
    && rm -f /usr/local/bin/npm /usr/local/bin/npx

COPY --from=build /app/public ./public
COPY --from=build --chown=nextjs:nodejs /app/.next/standalone ./
COPY --from=build --chown=nextjs:nodejs /app/.next/static ./.next/static
USER nextjs
EXPOSE 3000
CMD ["node", "server.js"]
