# Use the base image from Red Hat UBI Minimal 9.4
ARG DCM_BASE_IMAGE=registry.access.redhat.com/ubi9/ubi:9.4 
FROM ${DCM_BASE_IMAGE}

# Set environment variables
ENV DNF_FRONTEND=noninteractive
ENV CGO_ENABLED=1
ENV GOOS=linux
ENV GOARCH=amd64

RUN curl -fsSL https://golang.org/dl/go1.23.4.linux-amd64.tar.gz | tar -C /usr/local -xzf - && \
    ln -s /usr/local/go/bin/go /usr/bin/go

RUN mkdir -p /home/amd/tools/
ADD ./entrypoint.sh /home/amd/tools/entrypoint.sh
RUN chmod +x /home/amd/tools/entrypoint.sh

ENTRYPOINT ["/home/amd/tools/entrypoint.sh"]
