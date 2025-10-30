# Aura Terraform Example: production

This example provisions a production-leaning AWS stack with high availability and protections enabled:
- VPC across 3 Availability Zones with one NAT Gateway per AZ
- Security groups for Postgres and Redis (default allow from VPC; tighten to EKS node SGs as you mature)
- Amazon RDS for PostgreSQL with Multi-AZ, 14-day backups, deletion protection on
- Amazon ElastiCache for Redis with Multi-AZ and automatic failover
- Optional Amazon EKS managed cluster sized for HA (3+ nodes)

## Defaults tuned for production

- RDS: `multi_az = true`, `deletion_protection = true`, `backup_retention = 14`, `db.m6g.large`, 100GiB
- Redis: Multi-AZ + automatic failover, 2 nodes, `cache.m6g.large`
- VPC: 3 AZs with per-AZ NAT gateways (costlier but resilient)
- EKS: Enabled by default, 3 nodes (m6g.large)

Adjust sizes based on load, and enforce organization policies (KMS, tagging, SCPs, IAM boundaries, etc.).

## Usage

1. Create `terraform.tfvars`:

```
region      = "us-east-1"
db_username = "aura_admin"
db_password = "REPLACE_WITH_SECURE_PASSWORD"
# Optional: Change instance sizes/AZ count as needed
```

2. Initialize and apply:

```
terraform init
terraform apply
```

3. Map outputs to Helm or deployment env:

- `aurabackend_env` contains host/port/name/user for DB/Redis. Inject the DB password from AWS Secrets Manager or SSM Parameter Store.

## Security and reliability notes

- Replace VPC-wide ingress with security groups referencing the EKS node group SG to narrow blast radius.
- Consider using AWS KMS for database and EBS encryption with CMKs, and enable activity streams if required.
- Enable CloudWatch alarms and RDS/ElastiCache Enhanced Monitoring.
- For Redis, review cluster mode vs. replication group needs; adjust node counts accordingly.
