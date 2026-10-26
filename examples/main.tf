terraform {
  required_providers {
    outageme = { source = "marcioyoshida/outageme" }
  }
}

# OUTAGEME_API_KEY: an API key from the Keys tab of https://app.outage.me
provider "outageme" {}

# Prove you own the domain to unlock 1-minute, TCP and DNS checks on it.
resource "outageme_domain" "acme" {
  host = "acme.com"
}

# Publish the proof with any DNS provider, for example:
# resource "cloudflare_record" "outageme" {
#   zone_id = var.zone_id
#   name    = outageme_domain.acme.txt_name
#   type    = "TXT"
#   content = outageme_domain.acme.txt_value
# }

resource "outageme_domain_verification" "acme" {
  host = outageme_domain.acme.host
  # depends_on = [cloudflare_record.outageme]
}

resource "outageme_channel" "pager" {
  kind   = "pagerduty"
  target = var.pagerduty_routing_key
}

resource "outageme_check" "api" {
  name      = "Checkout API"
  url       = "https://api.acme.com/health"
  interval  = 1
  keyword   = "ok"
  locations = ["us-east-1", "eu-west-2", "ap-southeast-1"]
  deps      = ["Stripe", "AWS"]
  sla       = 99.9
  channels  = [outageme_channel.pager.id]
  depends_on = [outageme_domain_verification.acme]
}

resource "outageme_maintenance_window" "deploys" {
  label   = "Sunday deploys"
  days    = [6]
  at      = "02:00"
  minutes = 30
}

resource "outageme_status_page" "acme" {
  slug      = "acme"
  title     = "Acme status"
  checks    = [outageme_check.api.id]
  published = true
}

variable "pagerduty_routing_key" {
  type      = string
  sensitive = true
}
