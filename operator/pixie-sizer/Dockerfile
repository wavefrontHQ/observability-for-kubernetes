FROM gcr.io/distroless/static:latest
ARG TARGETPLATFORM
#   nobody:nobody
USER 65534:65534
WORKDIR /
COPY $TARGETPLATFORM .

ENTRYPOINT ["/pixie-sizer"]
