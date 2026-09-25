# AWS myApplications: registering inkwell as an AppRegistry application gives
# it a console dashboard (Console → myApplications → inkwell) with cost and
# usage, resources, alarms and security findings for everything carrying the
# application's `awsApplication` tag. AWS activates that tag for cost
# allocation automatically, so the cost widget needs no Billing setup.
resource "aws_servicecatalogappregistry_application" "inkwell" {
  name        = local.name
  description = "inkwell: encrypted journaling app (web, API, auth, data)"
}

locals {
  # { awsApplication = <application ARN> } — merged into each resource's tags.
  # Not in provider default_tags: the provider can't depend on a resource it
  # creates itself.
  app_tag = aws_servicecatalogappregistry_application.inkwell.application_tag
}
