ARG MCBACKUP_VER=0.2.1

FROM golang:alpine3.11

ARG GO111MODULE=on
WORKDIR /go/src/github.com/spritsail/mcbackup

RUN apk --no-cache add gcc musl-dev zfs-dev git

ADD go.mod go.sum ./
RUN go mod download

ARG MCBACKUP_VER
ADD . ./
RUN go build \
        -v \
        -ldflags="-w -s -X 'main.Version=$MCBACKUP_VER'" \
        -o /mcbackup

# ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

FROM spritsail/alpine:3.11

ARG MCBACKUP_VER
LABEL maintainer="Spritsail <mcbackup@spritsail.io>" \
      org.label-schema.vendor="Spritsail" \
      org.label-schema.name="mcbackup" \
      org.label-schema.url="https://git.spritsail.io/spritsail/mcbackup" \
      org.label-schema.description="Automatic Minecraft server backup utility" \
      org.label-schema.version=${MCBACKUP_VER}

# Install runtime dependencies
RUN apk --no-cache add zfs-libs

COPY --from=0 /mcbackup /usr/bin

ENV SUID=0 SGID=0
ENV BACKUP_DIRECTORY=/backups
WORKDIR $BACKUP_DIRECTORY
ENTRYPOINT ["/sbin/tini", "--", "su-exec", "-e", "/usr/bin/mcbackup"]
