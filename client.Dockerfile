FROM golang:alpine as builder
COPY src/ /go/src/
WORKDIR /go/src/client
RUN go build -o client

FROM golang:alpine
COPY --from=builder /go/src/client /root
ENTRYPOINT [ "/root/client" ]