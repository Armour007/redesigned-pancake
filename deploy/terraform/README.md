# Aura Terraform (cloud scaffolding)

This directory contains reusable modules to provision cloud infrastructure for Aura:

- modules/rds-postgres: RDS PostgreSQL instance
- modules/elasticache-redis: Redis replication group (TLS)
- modules/eks: EKS cluster (wrapper around terraform-aws-modules/eks)

## Example: Provision DB + Redis and feed Helm values

```hcl
module "db" {
  source                 = "./modules/rds-postgres"
  name                   = "aura-db"
  engine_version         = "15.5"
  instance_class         = "db.t3.micro"
  allocated_storage      = 20
  username               = var.db_username
  password               = var.db_password
  db_name                = "aura_db"
  vpc_id                 = var.vpc_id
  subnet_ids             = var.private_subnet_ids
  security_group_ids     = [aws_security_group.db.id]
}

module "redis" {
  source             = "./modules/elasticache-redis"
  name               = "aura-redis"
  engine_version     = "7.0"
  node_type          = "cache.t3.micro"
  subnet_ids         = var.private_subnet_ids
  security_group_ids = [aws_security_group.redis.id]
}

output "helm_values" {
  value = {
    env = {
      DB_HOST        = module.db.endpoint
      DB_PORT        = tostring(module.db.port)
      DB_USER        = module.db.username
      DB_PASSWORD    = var.db_password
      DB_NAME        = module.db.database
      AURA_REDIS_ADDR = "${module.redis.primary_endpoint}:${module.redis.port}"
    }
  }
}
```

You can feed `helm_values.env` into the Helm chart (see `deploy/helm/aura/values.yaml`) by setting `.Values.env`.

## Example: EKS cluster

```hcl
module "eks" {
  source              = "./modules/eks"
  name                = "aura-eks"
  version             = "1.29"
  vpc_id              = var.vpc_id
  private_subnet_ids  = var.private_subnet_ids
  node_instance_types = ["t3.medium"]
  desired_size        = 2
  min_size            = 1
  max_size            = 4
}
```

Apply with your preferred backend and variables.
