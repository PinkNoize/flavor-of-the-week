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
    "secretmanager.googleapis.com",
  ]
}

resource "random_id" "id" {
  byte_length = 8
}

# Backend
resource "google_storage_bucket" "tfstate_bucket" {
  name          = "${random_id.id.hex}-tfstate"
  location      = var.region
  force_destroy = true

  public_access_prevention = "enforced"
}

# Cloud Build Service account

resource "google_service_account" "cloudbuild_service_account" {
  account_id = "cloud-sa"
}

resource "google_project_iam_member" "act_as" {
  project = var.project
  role    = "roles/iam.serviceAccountUser"
  member  = "serviceAccount:${google_service_account.cloudbuild_service_account.email}"
}

resource "google_project_iam_member" "storage_admin" {
  project = var.project
  role    = "roles/storage.admin"
  member  = "serviceAccount:${google_service_account.cloudbuild_service_account.email}"
}

resource "google_project_iam_member" "cloudfunctions_admin" {
  project = var.project
  role    = "roles/cloudfunctions.admin"
  member  = "serviceAccount:${google_service_account.cloudbuild_service_account.email}"
}

resource "google_project_iam_member" "appengine_deployer" {
  project = var.project
  role    = "roles/appengine.deployer"
  member  = "serviceAccount:${google_service_account.cloudbuild_service_account.email}"
}

resource "google_project_iam_member" "appengine_creator" {
  project = var.project
  role    = "roles/appengine.appCreator"
  member  = "serviceAccount:${google_service_account.cloudbuild_service_account.email}"
}

resource "google_project_iam_member" "run_admin" {
  project = var.project
  role    = "roles/run.admin"
  member  = "serviceAccount:${google_service_account.cloudbuild_service_account.email}"
}

resource "google_project_iam_member" "iam_role_admin" {
  project = var.project
  role    = "roles/iam.roleAdmin"
  member  = "serviceAccount:${google_service_account.cloudbuild_service_account.email}"
}

resource "google_project_iam_member" "iam_service_acc_creator" {
  project = var.project
  role    = "roles/iam.serviceAccountCreator"
  member  = "serviceAccount:${google_service_account.cloudbuild_service_account.email}"
}

resource "google_project_iam_member" "iam_service_acc_deleter" {
  project = var.project
  role    = "roles/iam.serviceAccountDeleter"
  member  = "serviceAccount:${google_service_account.cloudbuild_service_account.email}"
}

resource "google_project_iam_member" "pubsub_admin" {
  project = var.project
  role    = "roles/pubsub.admin"
  member  = "serviceAccount:${google_service_account.cloudbuild_service_account.email}"
}

resource "google_project_iam_member" "cloudscheduler_admin" {
  project = var.project
  role    = "roles/cloudscheduler.admin"
  member  = "serviceAccount:${google_service_account.cloudbuild_service_account.email}"
}

resource "google_project_iam_member" "service_usage_admin" {
  project = var.project
  role    = "roles/serviceusage.serviceUsageAdmin"
  member  = "serviceAccount:${google_service_account.cloudbuild_service_account.email}"
}

resource "google_project_iam_member" "logs_writer" {
  project = var.project
  role    = "roles/logging.logWriter"
  member  = "serviceAccount:${google_service_account.cloudbuild_service_account.email}"
}

resource "google_secret_manager_secret_iam_member" "build_member" {
  project   = var.project
  secret_id = google_secret_manager_secret.discord_api.id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.cloudbuild_service_account.email}"
}

# Discord API secret

resource "google_secret_manager_secret" "discord_api" {
  secret_id = "discord-api-${random_id.id.hex}"

  replication {
    user_managed {
      replicas {
        location = var.region
      }
    }
  }
}

# Prod trigger

resource "google_cloudbuild_trigger" "prod_trigger" {
  name     = "prod-flavor-of-the-week-${random_id.id.hex}"
  location = var.region

  repository_event_config {
    repository = "projects/${var.project}/locations/${var.region}/connections/${var.repo_connection}/repositories/${var.repo_name}"
    push {
      branch = "^main$"
    }
  }
  included_files = [
    "infra/**",
    "functions/**",
    "deploy-commands-function/**",
    "cloudbuild.yaml"
  ]

  substitutions = {
    _TFSTATE_BUCKET     = google_storage_bucket.tfstate_bucket.name,
    _DISCORD_APP_ID     = var.discord_app_id,
    _DISCORD_PUBLIC_KEY = var.discord_public_key,
    _DISCORD_SECRET_ID  = google_secret_manager_secret.discord_api.id,
  }

  service_account = google_service_account.cloudbuild_service_account.id
  filename        = "cloudbuild.yaml"
  depends_on = [
    google_project_iam_member.act_as,
    google_project_iam_member.appengine_deployer,
    google_project_iam_member.appengine_creator,
    google_project_iam_member.cloudfunctions_admin,
    google_project_iam_member.iam_role_admin,
    google_project_iam_member.storage_admin,
    google_project_iam_member.run_admin,
    google_project_iam_member.iam_service_acc_creator,
    google_project_iam_member.iam_service_acc_deleter,
    google_project_iam_member.pubsub_admin,
    google_project_iam_member.cloudscheduler_admin,
    google_project_iam_member.service_usage_admin,
    google_project_iam_member.logs_writer,
    google_secret_manager_secret_iam_member.build_member,
  ]
}