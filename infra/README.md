# infra

Terraform for the inkwell AWS stack.

## Layout

- `bootstrap/` — one-shot: creates the S3 state bucket, DynamoDB lock table, and the GitHub Actions OIDC deploy role. Apply once with local state.
- `*.tf` (root) — the actual stack (KMS, DynamoDB, Cognito, Lambda, API Gateway, S3+CloudFront). CI assumes the deploy role from bootstrap to apply this.

## First-time setup

```sh
# 1. Bootstrap remote state (local state, one-shot)
cd bootstrap
terraform init
terraform apply
# note the `bucket` output

# 2. Init root with the bootstrap bucket
cd ..
terraform init \
  -backend-config="bucket=inkwell-tf-state-<ACCOUNT_ID>"

# 3. Apply
terraform apply
```

CI uses `-backend-config` from the `AWS_ACCOUNT_ID` env var; see `.github/workflows/terraform.yml`.

## Notes

- Lambda code is uploaded by the `api` workflow, not Terraform — TF only owns the function shell (role, env, name).
- CloudFront cert is the default `*.cloudfront.net`; add ACM + Route53 in `web.tf` when you bring a domain.
- The GH OIDC deploy role (`inkwell-gh-deploy`) uses `PowerUserAccess` plus a small IAM grant — tighten before any non-personal use. Defined in `bootstrap/main.tf`.
- The GitHub OIDC provider (`token.actions.githubusercontent.com`) is account-scoped; bootstrap consumes it via a data source rather than re-creating it.
