resource "aws_elasticache_subnet_group" "this" {
  name       = "${var.name}-redis-subnets"
  subnet_ids = var.subnet_ids
}

resource "aws_elasticache_replication_group" "this" {
  replication_group_id          = var.name
  replication_group_description = "${var.name} replication group"
  node_type                     = var.node_type
  engine                        = "redis"
  engine_version                = var.engine_version
  automatic_failover_enabled    = var.automatic_failover_enabled
  multi_az_enabled              = var.multi_az_enabled
  security_group_ids            = var.security_group_ids
  subnet_group_name             = aws_elasticache_subnet_group.this.name
  at_rest_encryption_enabled    = true
  transit_encryption_enabled    = true
}
