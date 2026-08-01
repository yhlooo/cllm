FROM scratch
ARG TARGETPLATFORM
COPY $TARGETPLATFORM/cllm /usr/bin/cllm
ENTRYPOINT ["/usr/bin/cllm"]
