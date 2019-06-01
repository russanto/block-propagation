FROM golang as builder
COPY src/ /go/src/
WORKDIR /go/src/server
RUN go get github.com/btcsuite/websocket && go build -o server
ENTRYPOINT [ "/go/server" ]