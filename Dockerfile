FROM golang:1.20.1-alpine as builder

WORKDIR /opt/go
COPY go.mod ./
COPY go.sum ./
COPY *.go ./

RUN go mod tidy
RUN CGO_ENABLED=1 GOEXPERIMENT=boringcrypto go build -o /usr/bin/size-openmetrics-reporter ./...

# FROM gcr.io/distroless/base-debian11:nonroot
# RUN apk update && apk add sh

# WORKDIR /opt/go
# COPY --from=builder /opt/go/app /opt/go/app
# ENTRYPOINT ["/opt/go/app"]