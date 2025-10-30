# RDS Postgres module

Creates an AWS RDS Postgres instance and subnet group.

Inputs:
- name (string)
- engine_version (string, default 15.5)
- instance_class (string, default db.t3.micro)
- allocated_storage (number, default 20)
- username (string)
- password (string)
- db_name (string)
- vpc_id (string)
- subnet_ids (list(string))
- security_group_ids (list(string), optional)
- backup_retention_period (number, default 7)
- multi_az (bool, default false)
- publicly_accessible (bool, default false)
- deletion_protection (bool, default true)

Outputs:
- endpoint, port, database, username

Example usage:

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
