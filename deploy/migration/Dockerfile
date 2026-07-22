FROM golang:1.26-alpine AS migrate-build

ARG GOLANG_MIGRATE_VERSION=v4.19.1
RUN test "$GOLANG_MIGRATE_VERSION" = "v4.19.1" \
    && CGO_ENABLED=0 go install -tags postgres \
        github.com/golang-migrate/migrate/v4/cmd/migrate@${GOLANG_MIGRATE_VERSION}

FROM alpine:3.21

RUN apk add --no-cache bash ca-certificates postgresql-client \
    && addgroup -S migration \
    && adduser -S -G migration migration

COPY --from=migrate-build /go/bin/migrate /usr/local/bin/migrate
COPY --chown=migration:migration deploy/migrate-op.sh /usr/local/bin/migrate-op
COPY --chown=migration:migration migrations/versioned /migrations/versioned
COPY --chown=migration:migration migrations/postgres-bootstrap /migrations/postgres-bootstrap
COPY --chown=migration:migration migrations/postgres /migrations/postgres

USER migration
ENTRYPOINT ["/usr/local/bin/migrate-op"]
