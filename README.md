# The Virtual Armory

A modern web application for firearms enthusiasts to catalog and manage their collection. The Virtual Armory provides a secure platform for users to document, organize, and track firearms with detailed specifications and features.

## Features

- Secure user authentication and authorization
- RBAC (Role-Based Access Control) for admin functionality
- Manufacturer, caliber, and weapon type management
- Subscription-based model with multiple tiers (free, monthly, yearly, lifetime)
- Promotion management for special subscription offers
- Stripe integration for payment processing
- CSRF protection across all forms
- Responsive design with Tailwind CSS

## Technology Stack

- **Backend**: Go with Gin/Gonic web framework
- **Frontend**: Tailwind CSS and HTMX
- **Authentication**: GoGuardian
- **Access Control**: Casbin for RBAC
- **Payment Processing**: Stripe
- **Template Engine**: Templ
- **Database**: PostgreSQL
- **Security**: Custom CSRF middleware

## Getting Started

These instructions will help you set up a local development environment for The Virtual Armory.

### Prerequisites

- Go 1.26 or later
- Docker (for PostgreSQL container)
- Node.js (for Tailwind CSS)

### Setup

1. Clone the repository
```bash
git clone https://github.com/hail2skins/armory.git
cd armory
```

2. Start the database container
```bash
make docker-run
```

3. Build the application
```bash
make build
```

4. Run the application
```bash
make run
```

The application should now be running at http://localhost:8080

### Development Workflow

Live reload the application during development:
```bash
make watch
```

## Makefile Commands

Run build with tests:
```bash
make all
```

Build the application:
```bash
make build
```

Run the application:
```bash
make run
```

Create database container:
```bash
make docker-run
```

Shutdown database container:
```bash
make docker-down
```

Run integration tests:
```bash
make itest
```

Run the test suite:
```bash
make test
```

Clean up binary from the last build:
```bash
make clean
```

## Project Structure

- `/cmd`: Application entry points
  - `/api`: API server
  - `/web`: Web server and templates
- `/internal`: Internal packages
  - `/controller`: HTTP handlers
  - `/middleware`: Custom middleware (CSRF, auth)
  - `/models`: Database models
  - `/services`: Business logic
- `/tests`: Test suites and utilities
- `/cmd/web/views`: Templ templates

## License

This project is licensed under the MIT License - see the LICENSE file for details.
