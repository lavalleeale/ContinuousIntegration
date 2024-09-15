FROM cypress/browsers:node-16.18.1-chrome-110.0.5481.96-1-ff-109.0-edge-110.0.1587.41-1

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

RUN apt-get update && apt-get install --no-install-recommends -y \
    docker-ce \
    docker-ce-cli \
    containerd.io \
    docker-buildx-plugin \
    redis \
    postgresql-client \
    iproute2

# Java fails without this
RUN  mkdir -p /usr/share/man/man1
# Install java
RUN apt-get update && apt-get install --no-install-recommends -y \
    default-jdk
ENV JAVA_HOME /usr/lib/jvm/java-8-openjdk-amd64
ENV PATH $JAVA_HOME/bin:$PATH
# Install python
RUN apt-get install --no-install-recommends -y \
    python3 \
    python3-pip \
    python3-setuptools \
    python3-wheel \
    python3-dev \
    python3-venv
# Download and install Go
RUN curl -OL https://golang.org/dl/go1.22.4.linux-amd64.tar.gz && \
    tar -C /usr/local -xzf go1.22.4.linux-amd64.tar.gz && \
    rm go1.22.4.linux-amd64.tar.gz

# Add Go to the PATH
ENV PATH $PATH:/usr/local/go/bin
# Install Ruby
RUN apt-get install --no-install-recommends -y \
    ruby-full
# Install C
RUN apt-get install --no-install-recommends -y \
    build-essential
# Install Rust
RUN curl https://sh.rustup.rs -sSf | sh -s -- -y
ENV PATH "/root/.cargo/bin:${PATH}"

ADD CA.pe[m] /usr/local/share/ca-certificates/registry.crt
RUN if [ -f "/usr/local/share/ca-certificates/registry.crt" ]; then \
    chmod 644 /usr/local/share/ca-certificates/registry.crt && update-ca-certificates; \
    fi


# Make directory for transferring files from previous stages
RUN mkdir -p /neededFiles