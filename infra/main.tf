terraform {
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "6.2.0"
    }
  }
}

provider "google" {
  project = var.project
  region  = var.region
}

module "project-services" {
  source  = "terraform-google-modules/project-factory/google//modules/project_services"
  version = "~> 17.0"

  project_id = var.project

  activate_apis = [
    "cloudbuild.googleapis.com",
    "iam.googleapis.com",
    "cloudresourcemanager.googleapis.com",
    "appengine.googleapis.com",
    "cloudfunctions.googleapis.com",
    "run.googleapis.com",
  ]
}

resource "random_id" "id" {
  byte_length = 8
}

resource "google_storage_bucket" "sources" {
  name                        = "${random_id.id.hex}-gcf-source"
  location                    = var.region
  uniform_bucket_level_access = true
}

# Enable Firestore
resource "google_app_engine_application" "app" {
  project       = var.project
  location_id   = var.region
  database_type = "CLOUD_FIRESTORE"
}

# Command Pub/Sub

# Command Function