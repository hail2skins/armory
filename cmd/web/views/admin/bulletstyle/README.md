# Bullet Style Admin Views

## Overview
This directory contains the user interface components for managing bullet styles in The Virtual Armory's admin section. Bullet styles represent different projectile types used in ammunition (e.g., FMJ, JHP, SP). These views implement the UI for the CRUD operations provided by the bullet style controller.

## Template Files
The implementation uses the [Templ](https://github.com/a-h/templ) templating library for Go, which provides type-safe templates with Go code integration.

The following template files are included:

1. `index.templ` - Displays a list of all bullet styles with actions
2. `new.templ` - Form for creating a new bullet style
3. `show.templ` - Detail view for a specific bullet style
4. `edit.templ` - Form for editing an existing bullet style

## Design Patterns

### MVC Architecture
- **Model**: Data structures are defined in `internal/models/bullet_style.go`
- **View**: Templates in this directory handle presentation
- **Controller**: Logic is implemented in `internal/controller/admin_bullet_style.go`

### Form Handling
Forms implement:
- CSRF protection tokens
- Client-side validation
- Server-side validation with error messaging
- Redirection with flash messages after successful operations

### Data Flow
1. Controller fetches data from the database service
2. Data is packaged into structs from `cmd/web/views/data`
3. Templates render the data with Tailwind CSS styling
4. Forms submit back to the controller for processing

## UI Features

### Index Page
- Displays all bullet styles in a responsive table
- Shows ID, Type, Nickname, and Popularity
- Provides links to view, edit, and delete each bullet style
- Includes a button to create new bullet styles

### Create/Edit Forms
- Input validation with feedback
- Required field for Type
- Optional Nickname field for full names or descriptions
- Popularity field for sorting (higher values appear first in lists)
- Cancel option to return to the list
- Consistent styling with the admin interface

### Deletion
- Uses POST requests for deletion (not GET)
- Provides confirmation before deletion
- Includes handling for cascade deletion constraints

## Integration with HTMX
The views use HTMX for enhanced interactivity where applicable:
- Form submissions without full page reloads
- Inline editing capabilities
- Dynamic content loading

## Styling
- Uses Tailwind CSS for consistent styling
- Follows The Virtual Armory's design system
- Responsive design for all device sizes

## Fields

### Bullet Style Model
- **ID**: Unique identifier
- **Type**: Primary name of the bullet style (e.g., FMJ, JHP, SP)
- **Nickname**: Optional full name or description (e.g., "Full Metal Jacket")
- **Popularity**: Numeric value for sorting (higher values appear first)
- **CreatedAt**: Timestamp of creation
- **UpdatedAt**: Timestamp of last update

## Usage Example
To render the index view:

```go
func (c *AdminBulletStyleController) Index(ctx *gin.Context) {
    // Get bullet styles from database
    bulletStyles, _ := c.db.FindAllBulletStyles()
    
    // Prepare view data
    adminData := getAdminBulletStyleDataFromContext(ctx, "Bullet Styles", ctx.Request.URL.Path)
    adminData = adminData.WithBulletStyles(bulletStyles)
    
    // Render the template
    component := bulletstyle.Index(adminData)
    component.Render(ctx.Request.Context(), ctx.Writer)
}
```

## Future Enhancements
- Advanced filtering options
- Bulk actions for multiple bullet styles
- Search functionality
- Support for bullet style images or visual indicators
- Integration with ammunition components system 