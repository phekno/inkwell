# Named grouping of everything tagged Project=inkwell (Console → Resource
# Groups & Tag Editor → Saved resource groups → inkwell), filterable by
# Component. Replaces an AppRegistry application: AppRegistry closed to new
# accounts on 2026-07-30, and AWS points to tag-based Resource Groups instead.
resource "aws_resourcegroups_group" "inkwell" {
  name        = local.name
  description = "All inkwell resources - tag Project is inkwell"

  resource_query {
    type = "TAG_FILTERS_1_0"
    query = jsonencode({
      ResourceTypeFilters = ["AWS::AllSupported"]
      TagFilters = [{
        Key    = "Project"
        Values = [local.name]
      }]
    })
  }

  tags = { Component = "ops" }
}
