FROM ubuntu:20.04

RUN apt-get update && \
    apt-get install --no-install-recommends -y ca-certificates=20210119~20.04.2 && \
    mkdir -p /etc/kangal && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

COPY kangal /bin/kangal
COPY openapi.json /etc/kangal/

RUN chmod a+x /bin/kangal && \
    chmod -R a+r /etc/kangal

# Use nobody user + group
USER 65534:65534

EXPOSE 8080
ENTRYPOINT ["/bin/kangal"]

# just to have it
RUN ["/bin/kangal", "--version"]
