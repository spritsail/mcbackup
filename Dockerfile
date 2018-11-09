ARG MCBACKUP_VER=0.0.1

FROM golang:alpine
WORKDIR /go/src/github.com/spritsail/mcbackup
ADD . .
RUN apk --no-cache add gcc git musl-dev zfs-dev
RUN go get -d
ARG MCBACKUP_VER
RUN go build \
        -v \
        -ldflags="-w -s -X 'main.Version=$MCBACKUP_VER'" \
        -o /mcbackup

# ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

FROM spritsail/alpine:3.8

ARG MCBACKUP_VER
LABEL maintainer="Spritsail <mcbackup@spritsail.io>" \
      org.label-schema.vendor="Spritsail" \
      org.label-schema.name="mcbackup" \
      org.label-schema.url="https://git.spritsail.io/spritsail/mcbackup" \
      org.label-schema.description="Automatic Minecraft server backup utility" \
      org.label-schema.version=${MCBACKUP_VER}

# Install runtime dependencies
RUN apk --no-cache add zfs-libs

COPY --from=0 /mcbackup /

WORKDIR /backups
CMD ["/mcbackup"]
