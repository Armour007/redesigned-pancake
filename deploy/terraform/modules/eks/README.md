# EKS module (wrapper)

Wraps terraform-aws-modules/eks to create an EKS cluster with a default managed node group.

Inputs:
- name (string)
- version (string, default 1.29)
- vpc_id (string)
- private_subnet_ids (list(string))
- public_subnet_ids (list(string), optional)
- node_instance_types (list(string), default ["t3.medium"]) 
- desired_size/min_size/max_size

Outputs:
- cluster_name, cluster_endpoint, cluster_ca_certificate
