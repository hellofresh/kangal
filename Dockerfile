FROM ubuntu:20.10

RUN mkdir -p /etc/kangal

ADD kangal /bin/kangal
ADD openapi.json /etc/kangal/

RUN chmod a+x /bin/kangal && \
    chmod -R a+r /etc/kangal

# Use nobody user + group
USER 65534:65534

EXPOSE 8080
ENTRYPOINT ["/bin/kangal"]

# just to have it
RUN ["/bin/kangal", "--version"]
