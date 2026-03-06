FROM node:22-alpine

WORKDIR /app

COPY server/package.json server/package-lock.json* ./
RUN npm install --production=false

COPY server/prisma ./prisma
RUN npx prisma generate

COPY server/ .
RUN npx tsc

RUN npm prune --production

RUN addgroup -S appgroup && adduser -S appuser -G appgroup \
    && chown -R appuser:appgroup /app

RUN mkdir -p /guacd-drive && chmod 1777 /guacd-drive

USER appuser

EXPOSE 3001 3002

HEALTHCHECK --interval=10s --timeout=5s --retries=5 --start-period=30s \
    CMD wget -qO- http://localhost:3001/api/health || exit 1

CMD ["sh", "-c", "npx prisma migrate deploy && node dist/index.js"]
