#!/bin/sh
set -eu

resolver="${NGINX_RESOLVER:-}"
if [ -z "$resolver" ]; then
    resolver="$(awk '/^nameserver[[:space:]]+/ { print $2; exit }' /etc/resolv.conf)"
fi

if [ -z "$resolver" ]; then
    echo "could not determine nginx resolver from /etc/resolv.conf" >&2
    exit 1
fi

sed "s|\${NGINX_RESOLVER}|$resolver|g" \
    /etc/nginx/templates/default.conf.template \
    > /tmp/default.conf

exec nginx -g 'daemon off;'
