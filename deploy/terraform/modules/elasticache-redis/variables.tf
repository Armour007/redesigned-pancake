variable "name" { type = string }
variable "engine_version" { type = string, default = "7.0" }
variable "node_type" { type = string, default = "cache.t3.micro" }
variable "num_cache_clusters" { type = number, default = 1 }
variable "automatic_failover_enabled" { type = bool, default = false }
variable "multi_az_enabled" { type = bool, default = false }
variable "vpc_id" { type = string }
variable "subnet_ids" { type = list(string) }
variable "security_group_ids" { type = list(string), default = [] }
