FROM node:lts-alpine AS builder

WORKDIR /app
RUN apk add git && \
    git clone https://github.com/olivia-ai/olivia-ai.org /app


FROM node:lts-alpine

COPY --from=builder /app /app

ARG VUE_APP_URL=ws://localhost:8080

RUN npm install -g http-server
WORKDIR /app
RUN npm ci
RUN npm run build

EXPOSE $WEBPORT
CMD http-server dist -p $WEBPORT
