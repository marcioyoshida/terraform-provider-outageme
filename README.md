# Terraform provider for outage.me Monitor

Uptime checks on your own endpoints, run from up to 5 locations, plus alert channels,
maintenance windows, domain ownership and your public status page, as code
(ADR 013 M3-2, #225). It is built on the `/v1/monitor` REST API
(https://outage.me/openapi.json, published while Monitor is on).

| Resource | What it manages |
|---|---|
| `outageme_check` | An http, tcp or dns check, with its alert rules (`sla`, `channels`, `alert_after`, `realert_min`, `escalate_min`, `escalate_to`) |
| `outageme_channel` | email (an org member), Slack, Teams, signed webhook, PagerDuty, Opsgenie |
| `outageme_maintenance_window` | One-off (`start`/`end`) or weekly UTC (`days`/`at`/`minutes`) |
| `outageme_domain` | A domain to prove; exports `txt_name`/`txt_value` and `file_url`/`file_body` |
| `outageme_domain_verification` | Waits until the proof is found (like `aws_acm_certificate_validation`) |
| `outageme_status_page` | `https://outage.me/s/<slug>/`; destroying it unpublishes the page |

See `examples/main.tf`. Authentication: `api_key` or `OUTAGEME_API_KEY`; `org` manages
another organization you are a member of, at your role there.

Behaviour worth knowing:
- Channels, windows and domains have no update API: changing one replaces it. Channel
  targets and keys are write-only (the API never returns them), so Terraform can't see a
  change made in the app.
- Removing a rule field (`sla`, `channels`…) from a check clears it. Omit lists instead of
  writing `[]`.
- Until a domain is verified, checks on it are limited to a plain GET on port 80/443 every
  5 minutes or slower, at most 3; make such checks `depends_on` the verification.

## Develop

    go build -o ~/.terraform.d/dev/terraform-provider-outageme .
    go test ./...

and point Terraform at it with a `dev_overrides` block in `~/.terraformrc`.

## Release

Push a `vX.Y.Z` tag: `.github/workflows/release.yml` runs GoReleaser, signs the checksums
with the provider's release GPG key (repository secrets `GPG_PRIVATE_KEY`, `PASSPHRASE`)
and creates the GitHub release the Terraform Registry picks up.

Source of truth: this provider is developed in the outage.me monorepo
(`tools/terraform-provider-outageme`) and copied here for releases.
