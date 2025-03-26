# Sitemap Generator

The Sitemap Generator is a living, self-updating component that automatically generates a `sitemap.xml` file for the Virtual Armory application. This enhances the SEO (Search Engine Optimization) by providing search engines with information about the structure of the site.

## Features

- **Automatic Route Discovery**: Automatically extracts all GET routes from the Gin router
- **Self-Updating**: Regenerates every hour to include any new routes
- **Customizable URLs**: Add custom URLs with specific change frequency, priority, and last modification date
- **Route Filtering**: Exclude private or internal routes using regex patterns
- **Robots.txt Integration**: Automatically creates a robots.txt file that points to the sitemap

## How It Works

The sitemap generator:

1. Extracts all GET routes from the Gin router
2. Filters out excluded routes (admin pages, private routes, etc.)
3. Generates an XML sitemap according to the [Sitemaps XML format](https://www.sitemaps.org/protocol.html)
4. Caches the generated XML to avoid unnecessary regeneration
5. Serves the sitemap.xml and robots.txt files

## Implementation

The sitemap functionality is implemented in several parts:

1. **`internal/sitemap/sitemap.go`**: Core generator that handles route extraction and XML generation
2. **`internal/controller/sitemap_controller.go`**: Controller that integrates the generator into the web application
3. **Routes integration**: Added in `internal/server/routes.go`

## Usage

The sitemap is automatically enabled and requires no manual intervention. It will:

- Be available at `/sitemap.xml`
- Include all GET routes that are not excluded by patterns
- Generate a robots.txt file at `/robots.txt` that points to the sitemap

## Customization

### Adding Custom URLs

You can add custom URLs with specific metadata by accessing the sitemap controller and using the `AddURL` method:

```go
sitemapController.generator.AddURL("/custom-page", sitemap.URLOptions{
    ChangeFreq: "weekly",
    Priority:   0.8,
    LastMod:    "2023-01-01",
})
```

### Excluding Routes

By default, the following routes are excluded:

- Admin routes (`/admin/*`)
- Authentication routes (`/reset-password*`, `/verification-*`)
- API endpoints (`/api/*`)
- Owner-specific pages (`/owner/*`)

You can add additional exclusion patterns:

```go
sitemapController.generator.ExcludePattern("^/internal")
```

### Setting the Host

The site host is determined in the following order:

1. From the `SITE_HOST` environment variable
2. Falls back to the default value (`https://virtualarmory.co`)

To change the host, set the `SITE_HOST` environment variable:

```bash
export SITE_HOST="https://yourdomain.com"
```

## Testing

Tests for the sitemap functionality are available in:

- **`tests/unit/sitemap_test.go`**: Unit tests for sitemap generation

Run the tests with:

```bash
go test ./tests/unit -v
```

## Maintenance and Updates

The sitemap is "living" in that it:

1. Automatically includes all GET routes
2. Regenerates on a regular basis (hourly when accessed)
3. Requires no manual updates when new pages are added

This ensures that as the application grows and changes, the sitemap remains up-to-date without needing manual intervention.

## SEO Benefits

- **Improved Indexing**: Search engines can discover pages faster
- **Better Crawl Efficiency**: Search engines understand your site structure
- **Structured Metadata**: Specify priority and update frequency of pages
- **Mobile/Desktop Distinction**: Can specify alternate URLs for different devices

## Troubleshooting

If you find routes missing from the sitemap:

1. Check if they're excluded by a pattern
2. Ensure they're GET routes (POST, PUT, etc. are excluded)
3. Verify they don't contain path parameters (`:id`)
4. Manually add important URLs with `AddURL`

To view the current sitemap, navigate to `/sitemap.xml` in your browser. 