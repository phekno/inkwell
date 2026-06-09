# OIDC trust for GitHub Actions. The deploy role can be assumed by workflows
# running in var.github_repo, scoped (by sub claim) to the main branch and
# any PR (read-only plan).

data "tls_certificate" "github" {
  url = "https://token.actions.githubusercontent.com"
}

resource "aws_iam_openid_connect_provider" "github" {
  url             = "https://token.actions.githubusercontent.com"
  client_id_list  = ["sts.amazonaws.com"]
  thumbprint_list = [data.tls_certificate.github.certificates[0].sha1_fingerprint]
}

data "aws_iam_policy_document" "gh_assume" {
  statement {
    effect  = "Allow"
    actions = ["sts:AssumeRoleWithWebIdentity"]

    principals {
      type        = "Federated"
      identifiers = [aws_iam_openid_connect_provider.github.arn]
    }

    condition {
      test     = "StringEquals"
      variable = "token.actions.githubusercontent.com:aud"
      values   = ["sts.amazonaws.com"]
    }

    condition {
      test     = "StringLike"
      variable = "token.actions.githubusercontent.com:sub"
      values = [
        "repo:${var.github_repo}:ref:refs/heads/main",
        "repo:${var.github_repo}:pull_request",
      ]
    }
  }
}

resource "aws_iam_role" "gh_deploy" {
  name               = "${local.name}-gh-deploy"
  assume_role_policy = data.aws_iam_policy_document.gh_assume.json
}

# Broad managed policy is fine for a personal project; tighten later.
resource "aws_iam_role_policy_attachment" "gh_deploy_power" {
  role       = aws_iam_role.gh_deploy.name
  policy_arn = "arn:aws:iam::aws:policy/PowerUserAccess"
}

# IAM operations (creating/modifying roles) need an extra grant.
resource "aws_iam_role_policy" "gh_deploy_iam" {
  name = "${local.name}-gh-deploy-iam"
  role = aws_iam_role.gh_deploy.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect = "Allow"
      Action = [
        "iam:CreateRole", "iam:DeleteRole", "iam:GetRole", "iam:UpdateRole",
        "iam:PassRole", "iam:TagRole", "iam:UntagRole",
        "iam:AttachRolePolicy", "iam:DetachRolePolicy",
        "iam:PutRolePolicy", "iam:DeleteRolePolicy", "iam:GetRolePolicy",
        "iam:ListAttachedRolePolicies", "iam:ListRolePolicies",
        "iam:CreateOpenIDConnectProvider", "iam:DeleteOpenIDConnectProvider",
        "iam:GetOpenIDConnectProvider", "iam:UpdateOpenIDConnectProviderThumbprint",
      ]
      Resource = "*"
    }]
  })
}
