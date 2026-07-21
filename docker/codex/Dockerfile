FROM golang:1.26-bookworm AS go

FROM node:22-bookworm-slim

COPY --from=go /usr/local/go /usr/local/go
ENV PATH="/usr/local/go/bin:${PATH}"

RUN apt-get update \
    && apt-get install -y --no-install-recommends git ripgrep ca-certificates \
    && rm -rf /var/lib/apt/lists/*

RUN npm install --global @openai/codex

WORKDIR /workspace
ENTRYPOINT ["codex"]
