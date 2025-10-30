output "endpoint" { value = aws_db_instance.this.address }
output "port" { value = aws_db_instance.this.port }
output "database" { value = aws_db_instance.this.db_name }
output "username" { value = var.username }
