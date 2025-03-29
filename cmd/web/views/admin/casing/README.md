# Casing Admin Views

## Overview
This directory contains the user interface components for managing ammunition casings in The Virtual Armory's admin section. These views implement the UI for the CRUD operations provided by the casing controller.

## Template Files
The implementation uses the [Templ](https://github.com/a-h/templ) templating library for Go, which provides type-safe templates with Go code integration.

The following template files are included:

1. `index.templ` - Displays a list of all casings with actions
2. `new.templ` - Form for creating a new casing
3. `show.templ` - Detail view for a specific casing
4. `edit.templ` - Form for editing an existing casing

## Design Patterns

### MVC Architecture
- **Model**: Data structures are defined in `internal/models/casing.go`
- **View**: Templates in this directory handle presentation
- **Controller**: Logic is implemented in `internal/controller/admin_casing.go`

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
- Displays all casings in a responsive table
- Provides links to view, edit, and delete each casing
- Includes a button to create new casings
- Supports sorting and pagination for large datasets

### Create/Edit Forms
- Input validation with feedback
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

## Usage Example
To render the index view:

```go
func (c *AdminCasingController) Index(ctx *gin.Context) {
    // Get casings from database
    casings, _ := c.db.FindAllCasings()
    
    // Prepare view data
    adminData := getAdminCasingDataFromContext(ctx, "Casings", ctx.Request.URL.Path)
    adminData = adminData.WithCasings(casings)
    
    // Render the template
    component := casing.Index(adminData)
    component.Render(ctx.Request.Context(), ctx.Writer)
}
```

## Future Enhancements
- Advanced filtering options
- Bulk actions for multiple casings
- Search functionality
- Export to CSV/Excel options 