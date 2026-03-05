FROM node:22-alpine AS build

WORKDIR /app

COPY client/package.json client/package-lock.json* ./
RUN npm install

COPY client/ .
RUN npm run build

FROM nginx:alpine
COPY --from=build /app/dist /usr/share/nginx/html
COPY client/nginx.conf /etc/nginx/conf.d/default.conf

RUN chown -R nginx:nginx /var/cache/nginx /var/log/nginx /etc/nginx/conf.d \
    && touch /var/run/nginx.pid && chown nginx:nginx /var/run/nginx.pid

USER nginx

EXPOSE 80

HEALTHCHECK --interval=10s --timeout=5s --retries=3 --start-period=5s \
    CMD wget -qO- http://localhost:80/ || exit 1

CMD ["nginx", "-g", "daemon off;"]
