resource "aws_kms_key" "entries" {
  description             = "inkwell: wraps per-user entry DEKs"
  deletion_window_in_days = 7
  enable_key_rotation     = true
}

resource "aws_kms_alias" "entries" {
  name          = "alias/${local.name}-entries"
  target_key_id = aws_kms_key.entries.key_id
}
