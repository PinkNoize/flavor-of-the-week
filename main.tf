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
  account_id = "cloud-sa-${random_id.id.hex}"
}

resource "google_project_iam_custom_role" "cloudbuild_policy_role" {
  role_id     = "cloud_sa_policy_${random_id.id.hex}"
  title       = "CloudBuild FOW SA Custom Role"
  description = ""
  permissions = ["resourcemanager.projects.getIamPolicy",
  "resourcemanager.projects.setIamPolicy"]
}

resource "google_project_iam_member" "cloud_build_custom_role" {
  project = var.project
  role    = google_project_iam_custom_role.cloudbuild_policy_role.id
  member  = "serviceAccount:${google_service_account.cloudbuild_service_account.email}"
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

resource "google_project_iam_member" "datastore_owner" {
  project = var.project
  role    = "roles/datastore.owner"
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

resource "google_project_iam_member" "secret_admin" {
  project = var.project
  role    = "roles/secretmanager.admin"
  member  = "serviceAccount:${google_service_account.cloudbuild_service_account.email}"
}

resource "google_project_iam_member" "eventarc_admin" {
  project = var.project
  role    = "roles/eventarc.admin"
  member  = "serviceAccount:${google_service_account.cloudbuild_service_account.email}"
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
  name     = "prod-flavor-of-the-week-${var.branch}-${random_id.id.hex}"
  location = var.region

  repository_event_config {
    repository = "projects/${var.project}/locations/${var.region}/connections/${var.repo_connection}/repositories/${var.repo_name}"
    push {
      branch = "^${var.branch}$"
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
    _ENV                = var.branch,
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
    google_project_iam_member.secret_admin,
    google_project_iam_member.eventarc_admin,
    google_project_iam_member.cloud_build_custom_role,
    google_project_iam_member.datastore_owner,
  ]
}

# PR Trigger Resources

resource "google_service_account" "cloudbuild_pr_service_account" {
  account_id = "cloud-pr-sa-${random_id.id.hex}"
}

resource "google_project_iam_custom_role" "pr_role" {
  role_id     = "${random_id.id.hex}_terraform_pr_role"
  title       = "Flavor of the Week PR Role"
  permissions = ["storage.buckets.list", "storage.buckets.get"]
}

resource "google_project_iam_member" "pr_act_as" {
  project = var.project
  role    = "roles/iam.serviceAccountUser"
  member  = "serviceAccount:${google_service_account.cloudbuild_pr_service_account.email}"
}

resource "google_project_iam_member" "pr_logs_writer" {
  project = var.project
  role    = "roles/logging.logWriter"
  member  = "serviceAccount:${google_service_account.cloudbuild_pr_service_account.email}"
}

resource "google_storage_bucket_iam_member" "pr_tfstate_bucket" {
  bucket     = google_storage_bucket.tfstate_bucket.name
  role       = "roles/storage.objectViewer"
  member     = "serviceAccount:${google_service_account.cloudbuild_pr_service_account.email}"
  depends_on = [google_storage_bucket.tfstate_bucket]
}

resource "google_project_iam_member" "pr_role" {
  project = var.project
  role    = google_project_iam_custom_role.pr_role.name
  member  = "serviceAccount:${google_service_account.cloudbuild_pr_service_account.email}"
}

resource "google_cloudbuild_trigger" "pr_trigger" {
  name     = "pr-flavor-of-the-week-${var.branch}-${random_id.id.hex}"
  location = var.region

  repository_event_config {
    repository = "projects/${var.project}/locations/${var.region}/connections/${var.repo_connection}/repositories/${var.repo_name}"

    pull_request {
      branch          = "^${var.branch}$"
      invert_regex    = false
      comment_control = "COMMENTS_ENABLED"
    }
  }

  substitutions = {
    _TFSTATE_BUCKET     = google_storage_bucket.tfstate_bucket.name,
    _DISCORD_APP_ID     = var.discord_app_id,
    _DISCORD_PUBLIC_KEY = var.discord_public_key,
    _DISCORD_SECRET_ID  = google_secret_manager_secret.discord_api.id,
    _ENV                = var.branch,
  }

  service_account = google_service_account.cloudbuild_service_account.id
  filename        = "pr_cloudbuild.yaml"
  depends_on = [
    google_project_iam_member.pr_act_as,
    google_project_iam_custom_role.pr_role,
    google_project_iam_member.pr_role,
    google_project_iam_member.pr_logs_writer,
    google_storage_bucket_iam_member.pr_tfstate_bucket,
  ]
}
