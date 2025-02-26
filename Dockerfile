FROM golang:1.21-alpine AS builder

RUN apk --no-cache add ca-certificates=20241121-r1  && \
    update-ca-certificates

FROM scratch
USER nobody
# Use nobody user + group
USER nobody:nobody
# Copy nobody user
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group

COPY kangal /bin/kangal

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

EXPOSE 8080
ENTRYPOINT ["/bin/kangal"]

# just to have it
RUN ["/bin/kangal", "--version"]
