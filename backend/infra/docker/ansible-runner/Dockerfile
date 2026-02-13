FROM ubuntu:22.04

# Install Ansible, PostgreSQL client, and dependencies
RUN apt-get update && apt-get install -y \
    ansible \
    postgresql-client \
    curl \
    unzip \
    python3-pip \
    && rm -rf /var/lib/apt/lists/*

# Install AWS CLI
RUN curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "awscliv2.zip" \
    && unzip awscliv2.zip \
    && ./aws/install \
    && rm awscliv2.zip

WORKDIR /ansible

COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]
