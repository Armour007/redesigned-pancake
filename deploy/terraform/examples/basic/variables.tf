variable "name" {
  type        = string
  description = "Base name/prefix for resources"
  default     = "aura"
}

variable "region" {
  type        = string
  description = "AWS region"
  default     = "us-east-1"
}

variable "vpc_cidr" {
  type        = string
  description = "VPC CIDR"
  default     = "10.0.0.0/16"
}

# RDS
variable "db_engine_version" { type = string, default = "15.5" }
variable "db_instance_class" { type = string, default = "db.t3.micro" }
variable "db_allocated_storage" { type = number, default = 20 }
variable "db_username" { type = string }
variable "db_password" { type = string, sensitive = true }
variable "db_name" { type = string, default = "aura" }
variable "db_backup_retention_days" { type = number, default = 7 }
variable "db_multi_az" { type = bool, default = false }
variable "db_deletion_protection" { type = bool, default = true }

# Redis
variable "redis_engine_version" { type = string, default = "7.0" }
variable "redis_node_type" { type = string, default = "cache.t3.micro" }
variable "redis_num_cache_clusters" { type = number, default = 1 }
variable "redis_automatic_failover" { type = bool, default = false }
variable "redis_multi_az" { type = bool, default = false }

# Optional EKS
variable "create_eks" { type = bool, default = false }
variable "eks_version" { type = string, default = "1.29" }
variable "eks_node_instance_types" { type = list(string), default = ["t3.medium"] }
variable "eks_desired_size" { type = number, default = 2 }
variable "eks_min_size" { type = number, default = 1 }
variable "eks_max_size" { type = number, default = 4 }
