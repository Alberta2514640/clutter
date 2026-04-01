FROM python:3.11-slim AS builder

RUN apt-get update && apt-get install -y --no-install-recommends \
    wget \
    git \
    unzip \
    ca-certificates \
    curl \
    && rm -rf /var/lib/apt/lists/*

RUN pip install --no-cache-dir \
    ansible-core==2.17.7 \
    awscli==1.33.44 \
    boto3==1.34.162 \
    botocore==1.34.162 \
    psycopg2-binary==2.9.9 \
    pyyaml==6.0.1 \
    jinja2==3.1.2

# Install the Ansible AWS collection for EC2 inventory metadata helpers/tasks.
RUN ansible-galaxy collection install amazon.aws:==8.2.1

# -------------------------------------------------------------------
FROM python:3.11-slim

# Install only runtime dependencies (including postgresql-client for psql)
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    postgresql-client \
    openssh-client \
    curl \
    && rm -rf /var/lib/apt/lists/*

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
