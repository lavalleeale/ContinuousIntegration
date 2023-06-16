FROM cypress/browsers:node-16.18.1-chrome-110.0.5481.96-1-ff-109.0-edge-110.0.1587.41-1

# Install go
COPY --from=golang:1.19 /usr/local/go/ /usr/local/go/

ENV PATH="/usr/local/go/bin:${PATH}"

#Install docker
RUN apt-get update && apt-get install --no-install-recommends -y \
    apt-transport-https \
    ca-certificates \
    curl \
    gnupg2 \
    software-properties-common
RUN curl -fsSL https://download.docker.com/linux/debian/gpg | apt-key add -
RUN apt-key fingerprint 0EBFCD88
RUN add-apt-repository \
    "deb [arch=amd64] https://download.docker.com/linux/debian \
    $(lsb_release -cs) \
    stable"
RUN apt-get update && apt-get install --no-install-recommends -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin

ADD CA.pem /usr/local/share/ca-certificates/registry.crt
RUN chmod 644 /usr/local/share/ca-certificates/registry.crt && update-ca-certificates

# Make directory for transferring files from previous stages
RUN mkdir -p /neededFiles