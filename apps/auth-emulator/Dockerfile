# syntax=docker/dockerfile:1.7
# Debian 13 (trixie) provides OpenJDK 21 in its stable repositories.
FROM node:24-trixie-slim

ARG FIREBASE_TOOLS_VERSION=15.26.0
ENV FIREBASE_TOOLS_VERSION=${FIREBASE_TOOLS_VERSION}
ENV FIREBASE_CLI_EXPERIMENTS=webframeworks

RUN apt-get update \
    && apt-get install -y --no-install-recommends openjdk-21-jre-headless ca-certificates \
    && java -version \
    && npm install --global "firebase-tools@${FIREBASE_TOOLS_VERSION}" --no-audit --no-fund \
    && firebase --version \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY --chown=node:node firebase.json entrypoint.sh ./
RUN chmod +x /app/entrypoint.sh \
    && mkdir -p /data \
    && chown node:node /data

EXPOSE 9099 4000
USER node
ENTRYPOINT ["/app/entrypoint.sh"]
