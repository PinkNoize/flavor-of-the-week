# Command Function

resource "google_cloudfunctions2_function" "command" {
  name        = "command-${random_id.id.hex}"
  location    = var.region
  description = "Command Function"

  build_config {
    runtime     = "go122"
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
    }
    service_account_email = google_service_account.cloud_func_service_account.email
  }

  event_trigger {
    trigger_region = var.region
    event_type     = "google.cloud.pubsub.topic.v1.messagePublished"
    pubsub_topic   = google_pubsub_topic.command_topic.id
    retry_policy   = "RETRY_POLICY_RETRY"
  }
}