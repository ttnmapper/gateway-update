FROM golang:latest as builder

WORKDIR /go-modules

COPY . ./

# Building using -mod=vendor, which will utilize the v
#RUN go get
#RUN go mod vendor
RUN CGO_ENABLED=0 GOOS=linux go build -v -mod=vendor -o app

#FROM alpine:3.8
FROM scratch
# We need ca certs installed otherwise we can't get https urls
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

WORKDIR /root/

COPY --from=builder /go-modules/app .
COPY conf.json .

CMD ["./app"]