FROM golang:1.16 as builder
WORKDIR /app
COPY ./ ./

RUN go mod download
RUN CGO_ENABLED=0 go build greeter_server/main.go

FROM scratch
COPY --from=builder /app/main /main
ENTRYPOINT ["/main"]