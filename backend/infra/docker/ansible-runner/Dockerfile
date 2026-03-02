FROM python:3.11-slim AS builder

RUN apt-get update && apt-get install -y --no-install-recommends \
    openssh-client \
    sshpass \
    wget \
    git \
    unzip \
    ca-certificates \
    curl \
    && rm -rf /var/lib/apt/lists/*

RUN pip install --no-cache-dir \
    ansible-core==2.16.0 \
    boto3>=1.34.0 \
    botocore>=1.34.0 \
    pyyaml>=6.0 \
    jinja2>=3.1.0

# Install the Ansible AWS collection (includes aws_ssm connection plugin)
RUN ansible-galaxy collection install amazon.aws community.aws

# Install the SSM Session Manager plugin (required for aws_ssm connection)
RUN curl -o /tmp/session-manager-plugin.deb \
    "https://s3.amazonaws.com/session-manager-downloads/plugin/latest/ubuntu_64bit/session-manager-plugin.deb" \
    && dpkg -i /tmp/session-manager-plugin.deb || true \
    && rm /tmp/session-manager-plugin.deb

# -------------------------------------------------------------------
FROM python:3.11-slim

# Install only runtime dependencies (including postgresql-client for psql)
RUN apt-get update && apt-get install -y --no-install-recommends \
    openssh-client \
    sshpass \
    ca-certificates \
    postgresql-client \
    curl \
    && rm -rf /var/lib/apt/lists/*

# Install SSM Session Manager plugin in runtime stage
RUN curl -o /tmp/session-manager-plugin.deb \
    "https://s3.amazonaws.com/session-manager-downloads/plugin/latest/ubuntu_64bit/session-manager-plugin.deb" \
    && dpkg -i /tmp/session-manager-plugin.deb || true \
    && rm /tmp/session-manager-plugin.deb

# Copy installed Python packages from builder
COPY --from=builder /usr/local/lib/python3.11/site-packages /usr/local/lib/python3.11/site-packages
COPY --from=builder /usr/local/bin /usr/local/bin

# Copy Ansible collections from builder
COPY --from=builder /root/.ansible /root/.ansible

# Create working directories
RUN mkdir -p /playbooks /tmp/ansible /var/log/ansible

# Copy scripts
COPY entrypoint.sh /entrypoint.sh
COPY generate_inventory.py /usr/local/bin/generate_inventory.py

RUN chmod +x /entrypoint.sh /usr/local/bin/generate_inventory.py

WORKDIR /playbooks
ENTRYPOINT ["/entrypoint.sh"]
