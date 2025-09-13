FROM golang:1.24-alpine AS build
ADD . /go/src/
WORKDIR /go/src
RUN go build notifier.go


FROM alpine:latest
COPY --from=build /go/src/notifier /usr/local/bin/notifier

CMD ["/usr/local/bin/notifier"]
