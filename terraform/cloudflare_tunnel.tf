# A remotely-managed Cloudflare Tunnel that serves the app at a stable custom
# hostname (e.g. stash.example.com) instead of the ephemeral *.trycloudflare.com
# quick tunnel. Everything is skipped when `domain` is empty.
locals {
  app_hostname     = var.domain != "" ? "${var.app_subdomain}.${var.domain}" : ""
  use_named_tunnel = local.app_hostname != ""
}

data "cloudflare_zone" "app" {
  count = local.use_named_tunnel ? 1 : 0

  filter = {
    name = var.domain
  }
}

resource "cloudflare_zero_trust_tunnel_cloudflared" "app" {
  count = local.use_named_tunnel ? 1 : 0

  account_id = var.cloudflare_account_id
  name       = "${var.project_name}-tunnel"
  config_src = "cloudflare"
}

resource "cloudflare_zero_trust_tunnel_cloudflared_config" "app" {
  count = local.use_named_tunnel ? 1 : 0

  account_id = var.cloudflare_account_id
  tunnel_id  = cloudflare_zero_trust_tunnel_cloudflared.app[0].id

  config = {
    ingress = [
      { hostname = local.app_hostname, service = "http://localhost:3333" },
      { service = "http_status:404" },
    ]
  }
}

resource "cloudflare_dns_record" "app" {
  count = local.use_named_tunnel ? 1 : 0

  zone_id = data.cloudflare_zone.app[0].zone_id
  name    = local.app_hostname
  content = "${cloudflare_zero_trust_tunnel_cloudflared.app[0].id}.cfargotunnel.com"
  type    = "CNAME"
  ttl     = 1
  proxied = true
}

data "cloudflare_zero_trust_tunnel_cloudflared_token" "app" {
  count = local.use_named_tunnel ? 1 : 0

  account_id = var.cloudflare_account_id
  tunnel_id  = cloudflare_zero_trust_tunnel_cloudflared.app[0].id
}
