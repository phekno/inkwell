#!/usr/bin/env bash
# Creates or updates the "inkwell" dashboard in AWS Billing and Cost
# Management (Console → Billing and Cost Management → Dashboards). The AWS
# provider has no resource for these dashboards yet, so this lives as a script.
#
# Requires the Project and Component cost allocation tags to be active (see
# ../README.md → "Tagging and cost reports"); until then the tag-filtered
# widgets show nothing.
#
# Usage: AWS_PROFILE=phekno infra/scripts/cost-dashboard.sh
set -euo pipefail

NAME=inkwell
export AWS_REGION=us-east-1 # Cost Management APIs are global, served from us-east-1

# Every widget is scoped to Project=inkwell.
filter='{"tags": {"key": "Project", "values": ["inkwell"], "matchOptions": ["EQUALS"]}}'

# widget TITLE WIDTH START GRANULARITY GROUPBY_JSON DISPLAY_JSON
widget() {
  local group=""
  [ "$5" != "null" ] && group=", \"groupBy\": [$5]"
  cat <<EOF
{
  "title": "$1",
  "width": $2,
  "height": 6,
  "configs": [{
    "queryParameters": {"costAndUsage": {
      "metrics": ["UnblendedCost"],
      "timeRange": {
        "startTime": {"type": "RELATIVE", "value": "$3"},
        "endTime": {"type": "RELATIVE", "value": "P0D"}
      },
      "granularity": "$4"$group,
      "filter": $filter
    }},
    "displayConfig": $6
  }]
}
EOF
}

bar='{"graph": {"UnblendedCost": {"visualType": "BAR"}}}'
table='{"table": {}}'
component='{"key": "Component", "type": "TAG"}'
service='{"key": "SERVICE", "type": "DIMENSION"}'

widgets="[
  $(widget "Monthly cost" 3 -P6M MONTHLY null "$bar"),
  $(widget "Monthly cost by component" 3 -P6M MONTHLY "$component" "$bar"),
  $(widget "Daily cost by component" 6 -P30D DAILY "$component" "$bar"),
  $(widget "Cost by service this quarter" 6 -P3M MONTHLY "$service" "$table")
]"

arn=$(aws bcm-dashboards list-dashboards \
  --query "dashboards[?name=='$NAME'].arn | [0]" --output text)

if [ "$arn" = "None" ] || [ -z "$arn" ]; then
  aws bcm-dashboards create-dashboard --name "$NAME" \
    --description "inkwell spend by component and service - tag Project: inkwell" \
    --widgets "$widgets" \
    --resource-tags '[{"key": "Project", "value": "inkwell"}, {"key": "Component", "value": "ops"}]' \
    --query arn --output text
else
  aws bcm-dashboards update-dashboard --arn "$arn" --name "$NAME" --widgets "$widgets" \
    --query arn --output text
fi
