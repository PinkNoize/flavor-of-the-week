# Poll scheduler
resource "google_cloud_scheduler_job" "poll-job" {
  name        = "poll-${random_id.id.hex}"
  description = "Trigger poll job"
  schedule    = "0 * * * *"
  pubsub_target {
    topic_name = google_pubsub_topic.poll_topic.id
    data       = base64encode("poll")
  }
}

# Poll Pub/Sub
resource "google_pubsub_topic" "poll_topic" {
  name = "poll-topic-${random_id.id.hex}"
}


# Poll function
resource "google_cloudfunctions2_function" "poll" {
  name        = "poll-${var.env}-${random_id.id.hex}"
  location    = var.region
  description = "Poll Function"

  build_config {
    runtime     = "go123"
    entry_point = "PollPubSub"
    source {
      storage_source {
        bucket = google_storage_bucket.sources.name
        object = google_storage_bucket_object.function-source.name
      }
    }
  }

  service_config {
    max_instance_count = 1
    available_memory   = "128Mi"
    timeout_seconds    = 500

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
    pubsub_topic   = google_pubsub_topic.poll_topic.id
    retry_policy   = "RETRY_POLICY_DO_NOT_RETRY"
  }
}
