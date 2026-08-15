FROM criyle/executorserver@sha256:aa3e913479c4ddeba47a0c2182fcb60fd7373b10fd4a31ff0f6893121e1b6e04

# Install essential compilers and runtime tools for the sandbox
RUN apt-get update && apt-get install -y wget gcc g++ python3 \
    && rm -rf /var/lib/apt/lists/*

# Download and install official Go 1.22.1 since go-judge config expects /usr/local/go/bin/go
RUN wget -qO go.tar.gz https://go.dev/dl/go1.22.1.linux-amd64.tar.gz \
    && tar -C /usr/local -xzf go.tar.gz \
    && rm go.tar.gz

ENV PATH="/usr/local/go/bin:${PATH}"
