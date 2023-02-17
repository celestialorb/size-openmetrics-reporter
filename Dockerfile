ARG BINARY_NAME="size-openmetrics-reporter"
FROM golang:1.20.1-alpine3.17 as builder

RUN apk add gcc musl-dev

WORKDIR /opt/go
COPY go.mod ./
COPY go.sum ./
COPY *.go ./

RUN go mod tidy
RUN CGO_ENABLED=1 GOEXPERIMENT=boringcrypto go build -o /usr/bin/${BINARY_NAME} ./...

FROM golang:1.20.1-alpine3.17

COPY --from=builder /usr/bin/${BINARY_NAME} /usr/bin/${BINARY_NAME}