# syntax=docker/dockerfile:1

ARG VARIANT=20
FROM mcr.microsoft.com/devcontainers/javascript-node:${VARIANT}

USER root

RUN apt-get update && \
    apt-get install -y --no-install-recommends \
    postgresql-client \
    redis-tools \
    ca-certificates \
    git \
    openssh-client \
    && rm -rf /var/lib/apt/lists/*

RUN corepack enable && corepack prepare pnpm@10.17.1 --activate

ENV PNPM_HOME=/home/node/.local/share/pnpm \
    PATH=/home/node/.local/share/pnpm:$PATH

USER node

WORKDIR /workspaces/kun-galgame-nuxt4
