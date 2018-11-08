FROM golang
WORKDIR /go/src/github.com/spritsail/mcbackup
ADD . .
RUN go get -d
RUN CGO_ENABLED=0 go build -v -ldflags='-w -s' -o /mcbackup

# ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

FROM spritsail/alpine:3.8

# Install runtime dependencies
RUN apk --no-cache add zfs

COPY --from=0 /mcbackup /

WORKDIR /backups
CMD ["/mcbackup"]
