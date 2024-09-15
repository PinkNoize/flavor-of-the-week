terraform {
  backend "gcs" {
    prefix = "flavor-of-the-week/env/pipeline"
  }
}