FROM golang:1.14 as build
WORKDIR /go/src/github.com/jukeizu/anipoll
COPY Makefile go.mod go.sum ./
RUN make deps
ADD . .
RUN make build-linux
RUN echo "nobody:x:100:101:/" > passwd

FROM scratch
COPY --from=build /go/src/github.com/jukeizu/anipoll/passwd /etc/passwd
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build --chown=100:101 /go/src/github.com/jukeizu/anipoll/bin/anipoll .
USER nobody
ENTRYPOINT ["./anipoll"]
