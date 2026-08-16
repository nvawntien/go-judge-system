FROM criyle/executorserver@sha256:aa3e913479c4ddeba47a0c2182fcb60fd7373b10fd4a31ff0f6893121e1b6e04

# Install compilers and runtimes used by AstraCode.
RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        wget \
        gcc \
        g++ \
        python3 \
        openjdk-17-jdk-headless \
    && rm -rf /var/lib/apt/lists/*

# Install Go 1.22.1 because the judge config expects /usr/local/go/bin/go.
RUN wget -qO go.tar.gz https://go.dev/dl/go1.22.1.linux-amd64.tar.gz \
    && tar -C /usr/local -xzf go.tar.gz \
    && rm go.tar.gz

ENV PATH="/usr/local/go/bin:${PATH}"