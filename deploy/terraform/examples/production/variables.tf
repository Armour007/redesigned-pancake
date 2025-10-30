variable "region"        { type = string  default = "us-east-1" }
variable "name"          { type = string  default = "aura-prod" }
variable "vpc_cidr"      { type = string  default = "10.1.0.0/16" }

# RDS
variable "db_engine_version"        { type = string  default = "15.5" }
variable "db_instance_class"        { type = string  default = "db.m6g.large" }
variable "db_allocated_storage"     { type = number  default = 100 }
variable "db_backup_retention_days" { type = number  default = 14 }
variable "db_multi_az"              { type = bool    default = true }
variable "db_deletion_protection"   { type = bool    default = true }
variable "db_username"              { type = string }
variable "db_password"              { type = string }
variable "db_name"                  { type = string  default = "aura" }

# Redis
variable "redis_engine_version"      { type = string  default = "7.0" }
variable "redis_node_type"           { type = string  default = "cache.m6g.large" }
variable "redis_num_cache_clusters"  { type = number  default = 2 }
variable "redis_automatic_failover"  { type = bool    default = true }
variable "redis_multi_az"            { type = bool    default = true }

# EKS
variable "create_eks"           { type = bool    default = true }
variable "eks_version"          { type = string  default = "1.29" }
variable "eks_node_instance_types" { type = list(string) default = ["m6g.large"] }
variable "eks_desired_size"     { type = number  default = 3 }
variable "eks_min_size"         { type = number  default = 3 }
variable "eks_max_size"         { type = number  default = 6 }
