from golang:1.6.0-alpine

ADD . /go/src/github.com/sdcoffey/gunviolencecounter
RUN go install github.com/sdcoffey/gunviolencecounter
ENTRYPOINT /go/bin/gunviolencecounter
