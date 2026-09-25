# infra

[OpenTofu](https://opentofu.org) for the inkwell AWS stack. Local: `brew install opentofu` and use `tofu` instead of `terraform`. Existing `.tf` files and the S3 backend lock file are compatible.

## Layout

- `bootstrap/` — one-shot: creates the S3 state bucket, DynamoDB lock table, and the GitHub Actions OIDC deploy role. Apply once with local state.
- `*.tf` (root) — the actual stack (KMS, DynamoDB, Cognito, Lambda, API Gateway, S3+CloudFront). CI assumes the deploy role from bootstrap to apply this.

## First-time setup

```sh
# 1. Bootstrap remote state (local state, one-shot)
cd bootstrap
tofu init
tofu apply
# note the `bucket` output

# 2. Init root with the bootstrap bucket
cd ..
tofu init \
  -backend-config="bucket=inkwell-tf-state-<ACCOUNT_ID>"

# 3. Apply
tofu apply
```

CI uses `-backend-config` from the `AWS_ACCOUNT_ID` env var; see `.github/workflows/tofu.yml`.

## Notes

- Lambda code is uploaded by the `api` workflow, not OpenTofu — IaC only owns the function shell (role, env, name).
- CloudFront cert is provisioned via ACM in `domain.tf` (DNS-validated through the phekno.com Route53 zone).
- The GH OIDC deploy role (`inkwell-gh-deploy`) uses `PowerUserAccess` plus a small IAM grant — tighten before any non-personal use. Defined in `bootstrap/main.tf`.
- The GitHub OIDC provider (`token.actions.githubusercontent.com`) is account-scoped; bootstrap consumes it via a data source rather than re-creating it.

## Tagging and cost reports

Every taggable resource carries, via `default_tags` on the provider (root and
`bootstrap/`):

| Tag | Value |
| --- | --- |
| `Project` | `inkwell` |
| `Environment` | `prod` |
| `ManagedBy` | `opentofu` |
| `Repo` | `github.com/phekno/inkwell` |

plus a per-resource `Component`: `web` (S3, CloudFront, ACM), `api` (Lambda,
its role and log group, API Gateway), `auth` (Cognito), `data` (DynamoDB,
KMS), `tfstate` (state bucket, lock table) and `ci` (GitHub deploy role).
Some resources can't be tagged (routes, CloudFront functions, KMS aliases,
Route53 records); their cost is negligible or rolls up into a tagged parent.

`bootstrap/` isn't applied by CI, so tag changes there need a local
`tofu apply` in `infra/bootstrap`.

**Tags don't show up in Cost Explorer until they're activated** as cost
allocation tags, once, in the paying account. A new tag key appears in
Billing up to 24h after a resource first carries it:

```sh
aws ce update-cost-allocation-tags-status --cost-allocation-tags-status \
  TagKey=Project,Status=Active TagKey=Component,Status=Active \
  TagKey=Environment,Status=Active

# Optional: apply them to up to 12 months of past costs too.
aws ce start-cost-allocation-tag-backfill --backfill-from 2025-10-01T00:00:00Z
```

Then in Cost Explorer, filter `Project = inkwell` and group by `Component`
(or by the `Tag: Component` dimension via `aws ce get-cost-and-usage`).
