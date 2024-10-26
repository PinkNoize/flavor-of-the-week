# Discord endpoint

resource "google_cloudfunctions2_function" "discord_endpoint" {
  name        = "discord-endpoint-${random_id.id.hex}"
  location    = var.region
  description = "Discord HTTP Endpoint"

  build_config {
    runtime     = "go122"
    entry_point = "DiscordFunctionEntry"
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
    timeout_seconds    = 10

    environment_variables = {
      PROJECT_ID     = var.project,
      COMMAND_TOPIC  = google_pubsub_topic.command_topic.name,
      DISCORD_PUBKEY = var.discord_public_key,
      ENV            = var.env,
    }
    secret_environment_variables {
      key        = "RAWG_TOKEN"
      project_id = var.project
      secret     = google_secret_manager_secret.rawg_api.secret_id
      version    = "latest"
    }
    service_account_email = google_service_account.cloud_func_service_account.email
  }
}

resource "google_cloud_run_service_iam_member" "member" {
  location = google_cloudfunctions2_function.discord_endpoint.location
  service  = google_cloudfunctions2_function.discord_endpoint.name
  role     = "roles/run.invoker"
  member   = "allUsers"
}

output "function_uri" {
  value = google_cloudfunctions2_function.discord_endpoint.service_config[0].uri
}
