variable "region" {
  type        = string
  description = "AWS region"
  default     = "us-east-1"
}

variable "enable_audit_bucket" {
  type        = bool
  description = "Create an S3 bucket for audit exports"
  default     = false
}

variable "audit_bucket_name" {
  type        = string
  description = "S3 bucket name for audit exports"
  default     = "aura-audit-exports-example"
}
