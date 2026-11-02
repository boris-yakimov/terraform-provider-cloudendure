terraform {
  required_providers {
    cloudendure = {
      source = "boris-yakimov/cloudendure"
    }
  }
}

# Credentials are best supplied through the environment:
#   export CLOUDENDURE_API_KEY="..."
#   export CLOUDENDURE_PROJECT_ID="..."
provider "cloudendure" {
  # api_key    = var.cloudendure_api_key
  # project_id = var.cloudendure_project_id
  # api_url    = "https://console.cloudendure.com/api/latest"
}
