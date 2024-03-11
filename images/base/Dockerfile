FROM cypress/browsers:node-16.18.1-chrome-110.0.5481.96-1-ff-109.0-edge-110.0.1587.41-1

# Install go
COPY --from=golang:1.20 /usr/local/go/ /usr/local/go/

ENV PATH="/usr/local/go/bin:${PATH}"

RUN curl -fsSL https://dl-ssl.google.com/linux/linux_signing_key.pub | tee /etc/apt/trusted.gpg.d/chrome.asc
RUN curl -fsSL https://download.docker.com/linux/debian/gpg | tee /etc/apt/trusted.gpg.d/docker.asc
#Install docker
RUN apt-get update && apt-get install --no-install-recommends -y \
    apt-transport-https \
    ca-certificates \
    curl \
    gnupg2 \
    software-properties-common
RUN add-apt-repository \
    "deb [arch=amd64] https://download.docker.com/linux/debian \
    $(lsb_release -cs) \
    stable"
RUN apt-get update && apt-get install --no-install-recommends -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin redis postgresql-client iproute2

ADD CA.pe[m] /usr/local/share/ca-certificates/registry.crt
RUN if [ -f "/usr/local/share/ca-certificates/registry.crt" ]; then \
    chmod 644 /usr/local/share/ca-certificates/registry.crt && update-ca-certificates; \
    fi


# Make directory for transferring files from previous stages
RUN mkdir -p /neededFiles