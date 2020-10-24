FROM alpine:3 as builder

RUN apk --no-cache update \
 && apk --no-cache upgrade \
 && apk --no-cache add git go \
 && mkdir -p /usr/src/www-util

COPY main.go /usr/src/www-util/
COPY go.mod  /usr/src/www-util/
COPY go.sum  /usr/src/www-util/

RUN cd /usr/src/www-util \
 && go get -d \
 && CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-extldflags "-static"' -o www-util

FROM scratch

COPY --from=builder /usr/src/www-util/www-util /

ENTRYPOINT ["/www-util"]
