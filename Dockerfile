
FROM golang:1.12-alpine AS builder

RUN apk add --no-cache dep git

WORKDIR $GOPATH/src/docker-tally

COPY Gopkg.toml Gopkg.lock ./
RUN dep ensure --vendor-only
COPY . ./
RUN CGO_ENABLED=0 GOOS=linux go build -a -o /docker-tally


FROM alpine:3.9
COPY --from=builder /docker-tally /
RUN mkdir /templates
COPY prometheus.tpl /templates

ENV TALLY_TEMPLATE=/templates/prometheus.tpl
ENV TALLY_OUTPUT=/output/config.yml

VOLUME ["/output"]
CMD ["/docker-tally"]


