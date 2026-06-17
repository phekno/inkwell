# inkwell

[![api](https://github.com/phekno/inkwell/actions/workflows/api.yml/badge.svg)](https://github.com/phekno/inkwell/actions/workflows/api.yml)
[![tui](https://github.com/phekno/inkwell/actions/workflows/tui.yml/badge.svg)](https://github.com/phekno/inkwell/actions/workflows/tui.yml)
[![web](https://github.com/phekno/inkwell/actions/workflows/web.yml/badge.svg)](https://github.com/phekno/inkwell/actions/workflows/web.yml)
[![tofu](https://github.com/phekno/inkwell/actions/workflows/tofu.yml/badge.svg)](https://github.com/phekno/inkwell/actions/workflows/tofu.yml)
[![codeql](https://github.com/phekno/inkwell/actions/workflows/codeql.yml/badge.svg)](https://github.com/phekno/inkwell/actions/workflows/codeql.yml)
[![checkov](https://github.com/phekno/inkwell/actions/workflows/checkov.yml/badge.svg)](https://github.com/phekno/inkwell/actions/workflows/checkov.yml)

A cloud-backed journaling app with a Go TUI (Bubble Tea), a Vue 3 web client, and a Go Lambda API on AWS. Entries are encrypted server-side using envelope encryption (KMS-wrapped per-user data keys).

- Web: https://journal.phekno.com
- API: https://journal.phekno.com/api (same-origin, path-based behind CloudFront)

## Layout

```
api/        Go Lambda HTTP API (entries CRUD, auth, crypto)
tui/        Bubble Tea TUI client
web/        Vue 3 + Vite + TS + Tailwind web client
infra/      OpenTofu (S3+DDB-backed state)
  bootstrap/  one-shot: creates the state bucket + lock table
.github/    CI/CD workflows
```

## AWS architecture

- API Gateway (HTTP API) → Lambda (Go) → DynamoDB (single-table)
- KMS CMK + per-user data keys, envelope-encrypted entry bodies
- Cognito User Pool for auth
- S3 + CloudFront for the web client; `/api/*` behavior fronts API Gateway via a CloudFront Function that strips the prefix
- ACM cert + Route53 record in the `phekno.com` hosted zone
- GitHub Actions deploys via OIDC-assumed IAM role (no static keys)

## Local dev

See each subdirectory's README for build/test/run instructions.
