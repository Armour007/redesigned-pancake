variable "name" { type = string }
variable "engine_version" { type = string, default = "15.5" }
variable "instance_class" { type = string, default = "db.t3.micro" }
variable "allocated_storage" { type = number, default = 20 }
variable "username" { type = string }
variable "password" { type = string }
variable "db_name" { type = string }
variable "vpc_id" { type = string }
variable "subnet_ids" { type = list(string) }
variable "security_group_ids" { type = list(string), default = [] }
variable "backup_retention_period" { type = number, default = 7 }
variable "multi_az" { type = bool, default = false }
variable "publicly_accessible" { type = bool, default = false }
variable "deletion_protection" { type = bool, default = true }
