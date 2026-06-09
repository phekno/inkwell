# inkwell

A cloud-backed journaling app with a Go TUI (Bubble Tea), a Vue 3 web client, and a Go Lambda API on AWS. Entries are encrypted server-side using envelope encryption (KMS-wrapped per-user data keys).

## Layout

```
api/        Go Lambda HTTP API (entries CRUD, auth, crypto)
tui/        Bubble Tea TUI client
web/        Vue 3 + Vite + TS + Tailwind web client
infra/      Terraform (S3+DDB-backed state)
  bootstrap/  one-shot: creates the state bucket + lock table
.github/    CI/CD workflows
```

## AWS architecture

- API Gateway (HTTP API) → Lambda (Go) → DynamoDB (single-table)
- KMS CMK + per-user data keys, envelope-encrypted entry bodies
- Cognito User Pool for auth
- S3 + CloudFront for the web client
- GitHub Actions deploys via OIDC-assumed IAM role (no static keys)

## Local dev

See each subdirectory's README for build/test/run instructions.
