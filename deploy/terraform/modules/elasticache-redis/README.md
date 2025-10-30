# ElastiCache Redis module

Creates a Redis replication group with TLS enabled.

Inputs:
- name (string)
- engine_version (string, default 7.0)
- node_type (string, default cache.t3.micro)
- num_cache_clusters (number, default 1)
- automatic_failover_enabled (bool, default false)
- multi_az_enabled (bool, default false)
- vpc_id (string)
- subnet_ids (list(string))
- security_group_ids (list(string), optional)

Outputs:
- primary_endpoint, reader_endpoint, port

Example usage:

module "redis" {
  source              = "./modules/elasticache-redis"
  name                = "aura-redis"
  engine_version      = "7.0"
  node_type           = "cache.t3.micro"
  subnet_ids          = var.private_subnet_ids
  security_group_ids  = [aws_security_group.redis.id]
}
