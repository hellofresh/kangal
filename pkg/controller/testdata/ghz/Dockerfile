FROM golang:1.19 as builder
WORKDIR /app
COPY ./ ./

RUN go mod download && \
    CGO_ENABLED=0 go build greeter_server/main.go

FROM scratch
COPY --from=builder /app/main /main
ENTRYPOINT ["/main"]
