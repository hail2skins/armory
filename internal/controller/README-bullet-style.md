# Bullet Style Admin Controller

## Overview
This document outlines the implementation of the Bullet Style admin controller for The Virtual Armory. The controller provides CRUD (Create, Read, Update, Delete) operations for managing bullet styles in the admin area of the application.

## Controller Implementation

The controller is implemented in `internal/controller/admin_bullet_style.go` with the following structure:

```go
type AdminBulletStyleController struct {
    db database.Service
}
```

It depends on the database service interface defined in `internal/database/database.go`.

## Endpoints and Actions

| HTTP Method | URL Pattern | Handler Function | Description |
|-------------|-------------|------------------|-------------|
| GET | `/admin/bullet_styles` | `Index` | List all bullet styles |
| GET | `/admin/bullet_styles/new` | `New` | Display form to create a new bullet style |
| POST | `/admin/bullet_styles` | `Create` | Create a new bullet style |
| GET | `/admin/bullet_styles/:id` | `Show` | Display details of a specific bullet style |
| GET | `/admin/bullet_styles/:id/edit` | `Edit` | Display form to edit a bullet style |
| POST | `/admin/bullet_styles/:id` | `Update` | Update a bullet style |
| POST | `/admin/bullet_styles/:id/delete` | `Delete` | Delete a bullet style |

## Authentication and Authorization

All routes are protected by:
1. Authentication middleware (requiring a logged-in user)
2. Casbin authorization middleware (requiring appropriate permissions)

The routes are configured in `internal/server/admin_routes.go` with the following permissions:
- `"bullet_styles", "read"` for Index and Show
- `"bullet_styles", "write"` for Create, Update, and Delete

## Handler Functions

### Index
Displays a list of all bullet styles.
- Fetches all bullet styles from the database using `FindAllBulletStyles()`
- Renders the `bulletstyle.Index` template

### New
Displays a form to create a new bullet style.
- Renders the `bulletstyle.New` template

### Create
Creates a new bullet style.
- Validates input from the form
- Creates a new `BulletStyle` model
- Calls `CreateBulletStyle()` on the database service
- Redirects to the index page with a success message

### Show
Displays details for a specific bullet style.
- Parses the ID from the URL
- Fetches the bullet style using `FindBulletStyleByID()`
- Renders the `bulletstyle.Show` template

### Edit
Displays a form to edit an existing bullet style.
- Parses the ID from the URL
- Fetches the bullet style using `FindBulletStyleByID()`
- Renders the `bulletstyle.Edit` template with the bullet style data

### Update
Updates an existing bullet style.
- Parses the ID from the URL
- Fetches the existing bullet style using `FindBulletStyleByID()`
- Updates the bullet style with form data
- Calls `UpdateBulletStyle()` on the database service
- Redirects to the index page with a success message

### Delete
Deletes a bullet style.
- Parses the ID from the URL
- Calls `DeleteBulletStyle()` on the database service
- Redirects to the index page with a success message

## Error Handling

Each handler function includes error handling for:
- Invalid URL parameters
- Database errors
- Validation errors

Errors are displayed to the user through the appropriate template.

## Testing

The controller is tested in `internal/controller/admin_bullet_style_test.go`, which includes tests for each handler function:

- `TestAdminBulletStyleIndex`
- `TestAdminBulletStyleNew`
- `TestAdminBulletStyleCreate`
- `TestAdminBulletStyleShow`
- `TestAdminBulletStyleEdit`
- `TestAdminBulletStyleUpdate`
- `TestAdminBulletStyleDelete`

The tests use:
- A test database
- Mock authentication
- Helper functions from `internal/testutils`

## Integration Points

The controller integrates with:
- **Database Service**: For data access operations
- **View Templates**: For rendering HTML
- **Authentication Middleware**: For user authentication
- **Casbin Middleware**: For authorization
- **CSRF Protection**: For security

## Data Flow

1. Client makes HTTP request
2. Middleware processes the request (auth, CSRF, etc.)
3. Controller handler is invoked
4. Controller fetches/manipulates data via database service
5. Controller passes data to templates
6. Templates render HTML response

## Example Usage

```go
// Register routes
bulletStyleGroup := adminGroup.Group("/bullet_styles")
bulletStyleGroup.GET("", adminBulletStyleController.Index)
bulletStyleGroup.GET("/new", adminBulletStyleController.New)
bulletStyleGroup.POST("", adminBulletStyleController.Create)
// Additional routes...

// Using the controller
adminBulletStyleController := controller.NewAdminBulletStyleController(service)
```

## Future Enhancements

- Implement soft delete and restore functionality
- Add filtering and searching capabilities
- Support for bulk operations
- Add pagination for large datasets 