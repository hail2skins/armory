# Stripe Services

This package provides services for integrating with Stripe, including payment processing, webhook handling, and IP filtering for enhanced security.

## Stripe IP Filtering

The IP filtering service adds protection to your webhook endpoints by ensuring that webhook requests come from legitimate Stripe IP addresses. This helps prevent unauthorized access to your webhook endpoints.

### How It Works

1. The service fetches IP ranges from all Stripe IP sources at application startup:
   - Webhook IPs: `https://stripe.com/files/ips/ips_webhooks.json`
   - API IPs: `https://stripe.com/files/ips/ips_api.json`
   - Armada/Gator IPs (files and other services): `https://stripe.com/files/ips/ips_armada_gator.json`

2. IP ranges are refreshed in the background at a configurable interval (default: 24 hours)
3. All requests to webhook endpoints are verified against these IP ranges
4. Requests from non-Stripe IPs are blocked with a 403 Forbidden response
5. The service provides fallback mechanisms to disable filtering in case of emergency

### Comprehensive Protection

Our implementation fetches and combines IP ranges from all Stripe services to provide maximum protection. We use the following approach:

1. **Multiple sources**: We fetch from all three Stripe IP sources to ensure we have complete coverage
2. **Fault tolerance**: If one source fails, we still use the IPs from the other sources
3. **Combined validation**: All IPs from all sources are used when validating incoming requests
4. **Proper CIDR handling**: We convert individual IPs to proper CIDR notation for validation

### Available Stripe IP Range Files

Stripe provides different IP range files for different services, and we use all of them:

- **Webhook IPs**: `https://stripe.com/files/ips/ips_webhooks.json`
  - IPs used for sending webhook events to your application

- **API IPs**: `https://stripe.com/files/ips/ips_api.json`
  - IPs used for Stripe's API infrastructure

- **Armada/Gator IPs**: `https://stripe.com/files/ips/ips_armada_gator.json`
  - IPs used for Stripe's file storage and other services

For more information, see the [Stripe IP Address documentation](https://docs.stripe.com/ips).

### Configuration

The IP filtering service is configured using environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `STRIPE_IP_FILTER_ENABLED` | Enable/disable IP filtering | `false` |
| `STRIPE_OVERRIDE_SECRET` | Secret for override header | `""` (empty) |

### Usage

#### Initialize and Start the Service

```go
// Create an HTTP client with a custom timeout if needed
httpClient := &http.Client{
    Timeout: 10 * time.Second,
}

// Create the IP filter service
ipFilterService := stripe.NewIPFilterService(httpClient)

// Start background refresh (requires a stop channel for graceful shutdown)
stopCh := make(chan struct{})
ipFilterService.StartBackgroundRefresh(stopCh)

// To stop background refresh (e.g., during graceful shutdown)
close(stopCh)
```

#### Apply Middleware

```go
// Apply the middleware to your router
router := gin.Default()
router.Use(ipFilterService.Middleware())

// Or apply selectively to webhook routes
webhookGroup := router.Group("/webhook")
webhookGroup.Use(ipFilterService.Middleware())
webhookGroup.POST("/stripe", stripeWebhookHandler)
```

#### Admin API Endpoints

The IP filtering service includes admin endpoints for monitoring and management:

1. **Status Check**: `GET /api/admin/stripe/ips`
   - Returns current status of IP ranges, including last update time and count

2. **Manual Refresh**: `POST /api/admin/stripe/ips/refresh`
   - Forces an immediate refresh of IP ranges

3. **Toggle Filtering**: `POST /api/admin/stripe/ips/toggle`
   - Enables or disables IP filtering

4. **IP Check**: `GET /api/admin/stripe/ip-check?ip=<IP_ADDRESS>`
   - Checks if a specific IP is in the allowed Stripe ranges

### Override Header

In case of emergency or for testing, you can bypass IP filtering by including a special header in your requests:

```
X-Stripe-Override: <STRIPE_OVERRIDE_SECRET>
```

Where `<STRIPE_OVERRIDE_SECRET>` is the value set in your environment variables.

### Command-Line Test Tool

The repository includes a command-line tool for testing IP ranges:

```bash
# Check if an IP is in Stripe's ranges
go run cmd/tools/ipfilter_test.go -ip 192.0.2.1

# Force refresh IP ranges
go run cmd/tools/ipfilter_test.go -refresh

# Show current status
go run cmd/tools/ipfilter_test.go -status

# Monitor IP ranges in background
go run cmd/tools/ipfilter_test.go -monitor
```

### Deployment Strategy

It's recommended to deploy the IP filtering in phases:

1. **Monitoring Phase**: 
   - Set `STRIPE_IP_FILTER_ENABLED=false`
   - Monitor logs for potential false positives
   - Use the admin API to check IP status

2. **Enforcement Phase**:
   - Set `STRIPE_IP_FILTER_ENABLED=true`
   - Monitor webhook reliability
   - Set up `STRIPE_OVERRIDE_SECRET` as a fallback

### Best Practices

1. **Always keep signature verification** as your primary security mechanism
2. **Set up monitoring alerts** for IP refresh failures
3. **Document the override procedure** for your team
4. **Periodically check** for IP range changes from Stripe

## Other Services

- **Webhook Handling**: Processing Stripe webhook events
- **Payment Processing**: Creating checkout sessions and handling payments 