terraform {
  required_version = ">= 1.5.0"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = ">= 5.0"
    }
  }
}

provider "aws" {
  region = var.region
}

# Skeleton: create an S3 bucket for audit exports (optional)
resource "aws_s3_bucket" "audit" {
  count  = var.enable_audit_bucket ? 1 : 0
  bucket = var.audit_bucket_name
}

output "audit_bucket_name" {
  value       = var.enable_audit_bucket ? aws_s3_bucket.audit[0].bucket : null
  description = "Name of the S3 bucket for audit exports"
}
