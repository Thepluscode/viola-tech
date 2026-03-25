FROM postgres:16-alpine
COPY scripts/dev/initdb/ /docker-entrypoint-initdb.d/
