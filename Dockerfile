FROM golang

ADD ./ /go/
WORKDIR /go/src/github.com/dearcj/od-corruption/
RUN go get -u github.com/golang/dep/...
RUN dep ensure

EXPOSE 80 443

RUN go install github.com/dearcj/od-corruption/

ENTRYPOINT ["/go/bin/od-corruption $OPT"]