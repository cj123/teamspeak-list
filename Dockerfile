FROM golang:latest

ADD . /go/src/github.com/cj123/teamspeak-list

WORKDIR /go/src/github.com/cj123/teamspeak-list

RUN go get .
RUN go build .

EXPOSE 2208

ENTRYPOINT /go/src/github.com/cj123/teamspeak-list/teamspeak-list
