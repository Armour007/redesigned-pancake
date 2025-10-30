variable "name" { type = string }
variable "version" { type = string, default = "1.29" }
variable "vpc_id" { type = string }
variable "private_subnet_ids" { type = list(string) }
variable "public_subnet_ids" { type = list(string), default = [] }
variable "node_instance_types" { type = list(string), default = ["t3.medium"] }
variable "desired_size" { type = number, default = 2 }
variable "min_size" { type = number, default = 1 }
variable "max_size" { type = number, default = 4 }
