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
    "eventarc.googleapis.com",
    "firestore.googleapis.com",
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

resource "google_firestore_index" "pool-search-index" {
  project    = var.project
  database   = "(default)"
  collection = "flavor-of-the-week"

  fields {
    field_path = "guild_id"
    order      = "ASCENDING"
  }

  fields {
    field_path = "search_name"
    order      = "ASCENDING"
  }
}

resource "google_firestore_index" "pool-type-search-index" {
  project    = var.project
  database   = "(default)"
  collection = "flavor-of-the-week"

  fields {
    field_path = "type"
    order      = "ASCENDING"
  }

  fields {
    field_path = "search_name"
    order      = "ASCENDING"
  }
}

resource "google_firestore_index" "pool-autocomplete-index" {
  project    = var.project
  database   = "(default)"
  collection = "flavor-of-the-week"

  fields {
    field_path = "guild_id"
    order      = "ASCENDING"
  }

  fields {
    field_path = "name"
    order      = "ASCENDING"
  }

  fields {
    field_path = "search_name"
    order      = "ASCENDING"
  }
}

resource "google_firestore_index" "nominations-search-index" {
  project    = var.project
  database   = "(default)"
  collection = "flavor-of-the-week"

  fields {
    field_path   = "nominations"
    array_config = "CONTAINS"
  }

  fields {
    field_path = "guild_id"
    order      = "ASCENDING"
  }

  fields {
    field_path = "search_name"
    order      = "ASCENDING"
  }
}

resource "google_firestore_index" "nominations-index" {
  project    = var.project
  database   = "(default)"
  collection = "flavor-of-the-week"

  fields {
    field_path = "guild_id"
    order      = "ASCENDING"
  }

  fields {
    field_path = "nominations_count"
    order      = "DESCENDING"
  }
}

resource "google_firestore_index" "random-1-index" {
  project    = var.project
  database   = "(default)"
  collection = "flavor-of-the-week"

  fields {
    field_path = "guild_id"
    order      = "ASCENDING"
  }

  fields {
    field_path = "random.num_1"
    order      = "ASCENDING"
  }
}

resource "google_firestore_index" "random-2-index" {
  project    = var.project
  database   = "(default)"
  collection = "flavor-of-the-week"

  fields {
    field_path = "guild_id"
    order      = "ASCENDING"
  }

  fields {
    field_path = "random.num_2"
    order      = "ASCENDING"
  }
}

# Cloud Functions role

resource "google_service_account" "cloud_func_service_account" {
  account_id   = "funcs-sa-${random_id.id.hex}"
  display_name = "Flavor of the Week Functions Account"
}

resource "google_project_iam_member" "firestore-iam" {
  project = var.project
  role    = "roles/datastore.user"
  member  = "serviceAccount:${google_service_account.cloud_func_service_account.email}"
}

# Add pub/sub publisher
resource "google_pubsub_topic_iam_member" "cloud_func_member" {
  project = google_pubsub_topic.command_topic.project
  topic   = google_pubsub_topic.command_topic.name
  role    = "roles/pubsub.publisher"
  member  = "serviceAccount:${google_service_account.cloud_func_service_account.email}"
}

resource "google_secret_manager_secret_iam_member" "cloud_func_member" {
  project   = var.project
  secret_id = var.discord_secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.cloud_func_service_account.email}"
}

resource "google_secret_manager_secret" "rawg_api" {
  secret_id = "discord-api-${random_id.id.hex}"

  replication {
    user_managed {
      replicas {
        location = var.region
      }
    }
  }
}

# Command Pub/Sub

resource "google_pubsub_topic" "command_topic" {
  name = "command-topic-${random_id.id.hex}"
}

# Functions source

data "archive_file" "default" {
  type        = "zip"
  output_path = "/tmp/functions-source.zip"
  source_dir  = "../functions/"
}
resource "google_storage_bucket_object" "function-source" {
  name   = "functions-source-${data.archive_file.default.output_sha256}.zip"
  bucket = google_storage_bucket.sources.name
  source = data.archive_file.default.output_path # Add path to the zipped function source code
}