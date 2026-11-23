FROM node:26-alpine AS build
WORKDIR /repo

COPY package.json package-lock.json ./
COPY packages/shared/package.json packages/shared/
COPY web/package.json web/
RUN npm ci

COPY packages packages
COPY web web
RUN npm run build --workspace @vcoach/web

FROM node:26-alpine
WORKDIR /repo
ENV NODE_ENV=production

COPY package.json package-lock.json ./
COPY packages/shared/package.json packages/shared/
COPY web/package.json web/
RUN npm ci --omit=dev

COPY --from=build /repo/web/dist ./web/dist
COPY web/server.mjs ./web/

WORKDIR /repo/web
EXPOSE 3000
CMD ["node", "server.mjs"]
