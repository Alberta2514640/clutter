#!/usr/bin/env python3
"""
Generate an Ansible inventory YAML file from EC2 instance IDs.

Queries the EC2 API for instance details and writes a YAML inventory
that uses direct SSH transport.

Usage:
    python3 generate_inventory.py \
        --instance-ids "i-0123456789abcdef0,i-0987654321fedcba0" \
        --region us-east-1 \
        --output /playbooks/inventory.yml \
        --aws-access-key-id AKIA... \
        --aws-secret-access-key ... \
        --aws-session-token ... \
        --ssh-private-key-file /playbooks/id_rsa \
        --host-address-source private_or_public
"""

import argparse
import sys
import yaml
import boto3
from botocore.exceptions import ClientError


def get_instance_details(instance_ids: list[str], region: str, session: boto3.Session) -> dict[str, dict]:
    """
    Query EC2 for instance details (ID, private/public IP, state).

    Returns:
        dict mapping instance_id -> {
            'private_ip': str,
            'public_ip': str,
            'state': str
        }
    """
    ec2 = session.client('ec2', region_name=region)

    instances = {}
    try:
        paginator = ec2.get_paginator('describe_instances')
        for page in paginator.paginate(InstanceIds=instance_ids):
            for reservation in page.get('Reservations', []):
                for instance in reservation.get('Instances', []):
                    instance_id = instance['InstanceId']
                    state = instance.get('State', {}).get('Name', 'unknown')

                    if state != 'running':
                        print(
                            f"[generate_inventory] WARNING: Instance {instance_id} is not running "
                            f"(state: {state})",
                            file=sys.stderr,
                        )
                        continue

                    instances[instance_id] = {
                        'private_ip': instance.get('PrivateIpAddress', ''),
                        'public_ip': instance.get('PublicIpAddress', ''),
                        'state': state,
                    }
    except ClientError as e:
        print(f"[generate_inventory] ERROR: Failed to describe instances: {e}", file=sys.stderr)
        sys.exit(1)

    return instances


def _select_host_address(details: dict, address_source: str) -> str:
    private_ip = details.get('private_ip', '')
    public_ip = details.get('public_ip', '')

    if address_source == 'public':
        return public_ip
    if address_source == 'private':
        return private_ip
    if address_source == 'private_or_public':
        return private_ip or public_ip
    # public_or_private
    return public_ip or private_ip


def generate_inventory(
    instances: dict[str, dict],
    remote_user: str,
    ssh_private_key_file: str,
    address_source: str,
) -> dict:
    """
    Build an Ansible inventory structure using direct SSH.
    """
    hosts = {}
    for instance_id, details in instances.items():
        ansible_host = _select_host_address(details, address_source)
        if not ansible_host:
            print(
                f"[generate_inventory] WARNING: Skipping {instance_id}; "
                f"no usable host address for source={address_source}",
                file=sys.stderr,
            )
            continue

        host_vars = {
            'ansible_host': ansible_host,
            'ansible_connection': 'ssh',
            'ansible_user': remote_user,
            'ansible_ssh_private_key_file': ssh_private_key_file,
            'ansible_ssh_common_args': (
                '-o IdentitiesOnly=yes '
                '-o ServerAliveInterval=30 '
                '-o ConnectTimeout=20'
            ),
        }
        hosts[instance_id] = host_vars

    group_vars = {
        'ansible_connection': 'ssh',
        'ansible_user': remote_user,
    }
    inventory = {
        'all': {
            'hosts': hosts,
            'vars': group_vars,
        },
    }

    return inventory


def main():
    parser = argparse.ArgumentParser(description='Generate Ansible inventory from EC2 instance IDs')
    parser.add_argument(
        '--instance-ids',
        required=True,
        help='Comma-separated list of EC2 instance IDs',
    )
    parser.add_argument(
        '--region',
        required=True,
        help='AWS region',
    )
    parser.add_argument(
        '--output',
        required=True,
        help='Output file path for the inventory YAML',
    )
    parser.add_argument(
        '--remote-user',
        default='ec2-user',
        help='SSH user for Ansible (default: ec2-user)',
    )
    parser.add_argument(
        '--aws-access-key-id',
        required=True,
        help='AWS access key ID for the client role',
    )
    parser.add_argument(
        '--aws-secret-access-key',
        required=True,
        help='AWS secret access key for the client role',
    )
    parser.add_argument(
        '--aws-session-token',
        required=True,
        help='AWS session token for the client role',
    )
    parser.add_argument(
        '--ssh-private-key-file',
        required=True,
        help='Path to SSH private key file used by Ansible',
    )
    parser.add_argument(
        '--host-address-source',
        choices=['public', 'private', 'public_or_private', 'private_or_public'],
        default='private_or_public',
        help='How to pick ansible_host from EC2 metadata (default: private_or_public)',
    )
    args = parser.parse_args()

    # Parse instance IDs
    instance_ids = [iid.strip() for iid in args.instance_ids.split(',') if iid.strip()]
    if not instance_ids:
        print('[generate_inventory] ERROR: No valid instance IDs provided', file=sys.stderr)
        sys.exit(1)

    print(f'[generate_inventory] Looking up {len(instance_ids)} instance(s) in {args.region}...')

    # Create a boto3 session with the client's assumed-role credentials for EC2 describe
    client_session = boto3.Session(
        aws_access_key_id=args.aws_access_key_id,
        aws_secret_access_key=args.aws_secret_access_key,
        aws_session_token=args.aws_session_token,
    )

    # Query EC2 using client credentials (their account)
    instances = get_instance_details(instance_ids, args.region, client_session)

    if not instances:
        print('[generate_inventory] ERROR: No reachable instances found', file=sys.stderr)
        sys.exit(1)

    print(f'[generate_inventory] Found {len(instances)} running instance(s)')

    # Generate and write inventory
    inventory = generate_inventory(
        instances,
        args.remote_user,
        args.ssh_private_key_file,
        args.host_address_source,
    )

    host_count = len(inventory.get('all', {}).get('hosts', {}))
    if host_count == 0:
        print('[generate_inventory] ERROR: No hosts with usable network addresses for SSH', file=sys.stderr)
        sys.exit(1)

    missing_hosts = sorted(set(instances.keys()) - set(inventory['all']['hosts'].keys()))
    if missing_hosts:
        print(
            '[generate_inventory] ERROR: Missing usable SSH address for instance(s): '
            + ','.join(missing_hosts),
            file=sys.stderr,
        )
        sys.exit(1)

    try:
        with open(args.output, 'w') as f:
            yaml.dump(inventory, f, default_flow_style=False)
        print(f'[generate_inventory] Inventory written to {args.output}')
    except OSError as e:
        print(f'[generate_inventory] ERROR: Failed to write inventory to {args.output}: {e}', file=sys.stderr)
        sys.exit(1)


if __name__ == '__main__':
    main()
