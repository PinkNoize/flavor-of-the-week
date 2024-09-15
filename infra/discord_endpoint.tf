# Discord endpoint

data "archive_file" "default" {
  type        = "zip"
  output_path = "/tmp/discord-endpoint-source.zip"
  source_dir  = "../functions/"
}
resource "google_storage_bucket_object" "discord-object" {
  name   = "discord-endpoint-source.zip"
  bucket = google_storage_bucket.sources.name
  source = data.archive_file.default.output_path # Add path to the zipped function source code
}

resource "google_cloudfunctions2_function" "default" {
  name        = "discord-endpoint-${random_id.id.hex}"
  location    = var.region
  description = "Discord HTTP Endpoint"

  build_config {
    runtime     = "go122"
    entry_point = "DiscordFunctionEntry"
    source {
      storage_source {
        bucket = google_storage_bucket.sources.name
        object = google_storage_bucket_object.discord-object.name
      }
    }
  }

  service_config {
    max_instance_count = 5
    available_memory   = "128Mi"
    timeout_seconds    = 10

    environment_variables = {
      DISCORD_PUBKEY = var.discord_public_key
    }
  }
}

resource "google_cloud_run_service_iam_member" "member" {
  location = google_cloudfunctions2_function.default.location
  service  = google_cloudfunctions2_function.default.name
  role     = "roles/run.invoker"
  member   = "allUsers"
}

output "function_uri" {
  value = google_cloudfunctions2_function.default.service_config[0].uri
}