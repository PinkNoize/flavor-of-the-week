# Command Function

locals {
  secret_id_split = split("/", var.discord_secret_id)
}

resource "google_cloudfunctions2_function" "command" {
  name        = "command-${random_id.id.hex}"
  location    = var.region
  description = "Command Function"

  build_config {
    runtime     = "go124"
    entry_point = "CommandPubSub"
    source {
      storage_source {
        bucket = google_storage_bucket.sources.name
        object = google_storage_bucket_object.function-source.name
      }
    }
  }

  service_config {
    max_instance_count = 5
    available_memory   = "128Mi"
    timeout_seconds    = 60

    environment_variables = {
      PROJECT_ID     = var.project,
      COMMAND_TOPIC  = google_pubsub_topic.command_topic.id,
      DISCORD_PUBKEY = var.discord_public_key,
      ENV            = var.env,
    }
    secret_environment_variables {
      key        = "DISCORD_TOKEN"
      project_id = var.project
      secret     = element(local.secret_id_split, length(local.secret_id_split) - 1)
      version    = "latest"
    }
    secret_environment_variables {
      key        = "RAWG_TOKEN"
      project_id = var.project
      secret     = google_secret_manager_secret.rawg_api.secret_id
      version    = "latest"
    }
    service_account_email = google_service_account.cloud_func_service_account.email
  }

  event_trigger {
    trigger_region = var.region
    event_type     = "google.cloud.pubsub.topic.v1.messagePublished"
    pubsub_topic   = google_pubsub_topic.command_topic.id
    retry_policy   = "RETRY_POLICY_DO_NOT_RETRY"
  }
}
