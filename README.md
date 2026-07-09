# Terraform Provider for NodePing

[![Go Version](https://img.shields.io/badge/Go-1.25-blue.svg)](https://golang.org/)
[![Terraform](https://img.shields.io/badge/Terraform-1.14+-purple.svg)](https://www.terraform.io/)
[![License](https://img.shields.io/badge/License-MPL--2.0-green.svg)](LICENSE)

A Terraform provider for managing [NodePing](https://nodeping.com/) monitoring resources.

**Timestamp**: 2026-01-27T08:33:00Z  
**Meta-prompt version**: v3.8

## Features

- **Contacts Management**: Create, read, update, and delete NodePing contacts with multiple notification addresses
- **Checks Management**: Full CRUD support for all 30+ NodePing check types
- **Multi-Account Support**: Manage resources across primary accounts and SubAccounts using provider aliases
- **Secure Authentication**: API token via configuration or environment variables
- **Rate Limiting**: Built-in rate limiting and retry logic

## Requirements

- [Terraform](https://www.terraform.io/downloads.html) >= 1.14
- [Go](https://golang.org/doc/install) >= 1.25 (for building from source)
- A [NodePing](https://nodeping.com/) account with API access

## Installation

### From Terraform Registry (Recommended)

```hcl
terraform {
  required_providers {
    nodeping = {
      source  = "ahmedbodi/nodeping"
      version = "~> 0.2"
    }
  }
}
```

### Building from Source

```bash
git clone https://gitlab.com/your-org/terraform-provider-nodeping.git
cd terraform-provider-nodeping
go build -o terraform-provider-nodeping
```

## Authentication

The provider requires a NodePing API token for authentication. You can obtain your token from the Account Settings section in the NodePing web interface.

### Option 1: Provider Configuration

```hcl
provider "nodeping" {
  api_token = var.nodeping_api_token
}
```

### Option 2: Environment Variables

```bash
export NODEPING_API_TOKEN="your-api-token"
```

### Configuration Precedence

1. Provider block configuration (highest priority)
2. Environment variables (fallback)

## Provider Configuration

```hcl
provider "nodeping" {
  # Required: API token for authentication
  # Can also be set via NODEPING_API_TOKEN environment variable
  api_token = "your-api-token"

  # Optional: SubAccount customer ID for managing SubAccount resources
  # Can also be set via NODEPING_CUSTOMER_ID environment variable
  customer_id = "201203232048C76FH"

  # Optional: API base URL (for testing)
  # Default: https://api.nodeping.com/api/1
  api_url = "https://api.nodeping.com/api/1"

  # Optional: Rate limiting (requests per second)
  # Default: 10
  rate_limit = 10

  # Optional: Retry configuration
  max_retries    = 3   # Maximum retry attempts
  retry_wait_min = 1   # Minimum wait between retries (seconds)
  retry_wait_max = 30  # Maximum wait between retries (seconds)
}
```

## Multi-Account Usage

Use provider aliases to manage resources across multiple accounts:

```hcl
# Primary account
provider "nodeping" {
  alias     = "primary"
  api_token = var.primary_token
}

# SubAccount
provider "nodeping" {
  alias       = "subaccount"
  api_token   = var.primary_token
  customer_id = "SUBACCOUNT_CUSTOMER_ID"
}

# Resources in primary account
resource "nodeping_contact" "primary_ops" {
  provider = nodeping.primary
  name     = "Primary Ops Team"
  # ...
}

# Resources in SubAccount
resource "nodeping_contact" "sub_ops" {
  provider = nodeping.subaccount
  name     = "SubAccount Ops Team"
  # ...
}
```

## Quick Start

### Create a Contact

```hcl
resource "nodeping_contact" "ops_team" {
  name     = "Operations Team"
  custrole = "notify"

  address {
    type    = "email"
    address = "ops@example.com"
  }

  address {
    type    = "sms"
    address = "+1-555-123-4567"
  }
}
```

### Create an HTTP Check

```hcl
resource "nodeping_check" "website" {
  type    = "HTTP"
  target  = "https://example.com"
  label   = "Example Website"
  enabled = true

  interval  = 5   # minutes
  threshold = 10  # seconds timeout
  sens      = 2   # rechecks before status change

  runlocations = ["nam"]  # North America
  tags         = ["production", "website"]

  notifications {
    contact_id = nodeping_contact.ops_team.id
    delay      = 0
    schedule   = "All"
  }
}
```

### Create an SSL Certificate Check

```hcl
resource "nodeping_check" "ssl_cert" {
  type        = "SSL"
  target      = "example.com"
  label       = "SSL Certificate Expiry"
  enabled     = true
  warningdays = 30
  servername  = "example.com"
}
```

### Create a Check with Notification Dependency

```hcl
# Primary check (e.g., router or core service)
resource "nodeping_check" "router" {
  type    = "PING"
  target  = "192.168.1.1"
  label   = "Edge Router"
  enabled = true
}

# Dependent check - notifications suppressed if router is down
resource "nodeping_check" "web_service" {
  type    = "HTTP"
  target  = "https://internal.example.com"
  label   = "Internal Web Service"
  enabled = true
  dep     = nodeping_check.router.id  # Suppress notifications if router check is failing
}
```

## Resources

### nodeping_contact

Manages a NodePing contact for receiving notifications.

**Example:**

```hcl
resource "nodeping_contact" "example" {
  name     = "John Doe"
  custrole = "notify"  # "edit", "view", or "notify"

  address {
    type    = "email"
    address = "john@example.com"
  }
}
```

**Attributes:**

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | No | Contact name/label |
| `custrole` | string | No | Permission role: `edit`, `view`, `notify` (default: `notify`) |
| `address` | block | No | Notification addresses (see below) |

**Address Block:**

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `type` | string | Yes | Address type: `email`, `sms`, `webhook`, `slack`, `pushover`, `pagerduty`, `voice` |
| `address` | string | Yes | Address value |
| `suppress_up` | bool | No | Suppress "up" notifications |
| `suppress_down` | bool | No | Suppress "down" notifications |
| `suppress_first` | bool | No | Suppress "first result" notifications |
| `suppress_diag` | bool | No | Suppress diagnostic notifications |
| `suppress_all` | bool | No | Suppress all notifications |
| `mute` | bool | No | Mute all notifications |
| `action` | string | No | HTTP method for webhooks |
| `headers` | map | No | HTTP headers for webhooks |
| `data` | string | No | Request body for webhooks |

### nodeping_check

Manages a NodePing monitoring check.

**Supported Check Types:**

`AGENT`, `AUDIO`, `CLUSTER`, `DOHDOT`, `DNS`, `FTP`, `HTTP`, `HTTPCONTENT`, `HTTPPARSE`, `HTTPADV`, `IMAP4`, `MONGODB`, `MTR`, `MYSQL`, `NTP`, `PGSQL`, `PING`, `POP3`, `PORT`, `PUSH`, `RBL`, `RDAP`, `RDP`, `REDIS`, `SIP`, `SMTP`, `SNMP`, `SPEC10DNS`, `SPEC10RDDS`, `SSH`, `SSL`, `WEBSOCKET`, `WHOIS`

**Common Attributes:**

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `type` | string | Yes | Check type |
| `target` | string | Conditional | Target URL/hostname/IP |
| `label` | string | No | Display label |
| `enabled` | bool | No | Enable the check |
| `interval` | float | No | Check interval in minutes |
| `threshold` | int | No | Timeout in seconds |
| `sens` | int | No | Rechecks before status change |
| `dep` | string | No | Check ID for notification dependency (suppresses notifications if dependent check is failing) |
| `runlocations` | list | No | Probe locations |
| `tags` | list | No | Tags for grouping |

## Data Sources

### nodeping_contact

Fetch a single contact by ID.

```hcl
data "nodeping_contact" "example" {
  id = "201205050153W2Q4C-BKPGH"
}
```

### nodeping_contacts

Fetch all contacts.

```hcl
data "nodeping_contacts" "all" {}
```

### nodeping_check

Fetch a single check by ID.

```hcl
data "nodeping_check" "example" {
  id = "201205050153W2Q4C-0J2HSIRF"
}
```

### nodeping_checks

Fetch all checks with optional filtering.

```hcl
data "nodeping_checks" "http_only" {
  type = "HTTP"
}
```

## Import

### Import a Contact

```bash
# Primary account
terraform import nodeping_contact.example 201205050153W2Q4C-BKPGH

# SubAccount
terraform import nodeping_contact.example CUSTOMER_ID:201205050153W2Q4C-BKPGH
```

### Import a Check

```bash
# Primary account
terraform import nodeping_check.example 201205050153W2Q4C-0J2HSIRF

# SubAccount
terraform import nodeping_check.example CUSTOMER_ID:201205050153W2Q4C-0J2HSIRF
```

## Security Considerations

### Sensitive Data

- **API Token**: Marked as sensitive; never logged or stored in state
- **Contact Addresses**: Email addresses and phone numbers are marked as sensitive
- **Passwords**: Check passwords (FTP, SSH, etc.) are marked as sensitive

### Terraform State

⚠️ **Warning**: Terraform state may contain sensitive data including:
- Contact email addresses and phone numbers
- Check target URLs and hostnames
- Webhook URLs and configurations

**Recommendations:**
- Use encrypted remote state backends (S3 with encryption, Terraform Cloud, etc.)
- Restrict access to state files
- Consider using `terraform state pull` with caution

### GDPR Compliance

This provider manages personal data (contact information). Ensure you:
- Have appropriate consent for storing contact data
- Document data processing activities
- Use `terraform destroy` to remove managed resources when no longer needed

## Development

### Building

```bash
go build -o terraform-provider-nodeping
```

### Testing

```bash
# Unit tests
go test ./...

# Acceptance tests (requires NODEPING_API_TOKEN)
TF_ACC=1 go test ./... -v
```

### Linting

```bash
golangci-lint run
gosec ./...
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run tests and linting
5. Submit a merge request

## License

This project is licensed under the Mozilla Public License 2.0 - see the [LICENSE](LICENSE) file for details.

## Support

- [NodePing Documentation](https://nodeping.com/documentation.html)
- [NodePing API Reference](https://nodeping.com/docs-api-overview.html)
- [Issue Tracker](https://gitlab.com/your-org/terraform-provider-nodeping/-/issues)

## Pinned Versions

| Component | Version |
|-----------|---------|
| Go | 1.25 |
| Terraform CLI | 1.14 |
| terraform-plugin-framework | latest compatible |
| terraform-plugin-testing | latest compatible |
