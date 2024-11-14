# Copyright (c) HashiCorp, Inc.

resource "stripe_webhook_endpoint" "example" {
  api_version = "2024-10-28.acacia"
  description = "Example webhook endpoint"
  enabled_events = [
    "charge.succeeded",
    "charge.failed",
  ]
  metadata = {
    foo = "bar"
  }
  url = "https://example.com/stripe-webhook"
}
