output "aurabackend_env" {
  description = "Environment variables for Aura backend Helm values or deployment"
  value = {
    AURA_DB_HOST  = module.rds.endpoint
    AURA_DB_PORT  = module.rds.port
    AURA_DB_NAME  = module.rds.database
    AURA_DB_USER  = module.rds.username
    # For security, don't output DB password directly; store in Secrets Manager/SSM and reference at deploy time.
    AURA_REDIS_HOST = module.redis.primary_endpoint
    AURA_REDIS_PORT = module.redis.port
  }
}

output "vpc_id" { value = module.vpc.vpc_id }
output "private_subnets" { value = module.vpc.private_subnets }
output "public_subnets" { value = module.vpc.public_subnets }

output "eks_cluster_name" {
  description = "EKS cluster name (when created)"
  value       = var.create_eks ? module.eks[0].cluster_name : null
}

output "eks_cluster_endpoint" {
  description = "EKS cluster endpoint (when created)"
  value       = var.create_eks ? module.eks[0].cluster_endpoint : null
}
