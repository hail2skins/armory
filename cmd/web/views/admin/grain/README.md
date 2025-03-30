# Grain Admin Views

## Overview
This directory contains the user interface components for managing ammunition grain weights in The Virtual Armory's admin section. These views implement the UI for the CRUD operations provided by the grain controller.

Grain weights are a critical property of ammunition, representing the weight of bullets in grains (a unit of measure where 1 grain = 0.0648 grams). The views in this directory provide a user-friendly interface for administrators to manage these data points.

## Files
1. `index.templ` - Displays a list of all grain weights with actions
2. `new.templ` - Form for creating a new grain weight
3. `show.templ` - Detail view for a specific grain weight
4. `edit.templ` - Form for editing an existing grain weight

## Architecture
The views in this directory follow the MVC pattern used throughout the application:
- **Model**: Data structures are defined in `internal/models/grain.go`
- **View**: Templates in this directory
- **Controller**: Logic is implemented in `internal/controller/admin_grain.go`

Database interactions are handled through the service layer defined in `internal/database/database.go`.

## Special Handling for "Other" Value
Grain weights usually have specific numerical values, but there's a special case where a weight of 0 should be displayed as "Other" in the UI. This is implemented in the template logic where:

```go
weightDisplay := fmt.Sprintf("%d", grain.Weight)
if grain.Weight == 0 {
    weightDisplay = "Other"
}
```

This display conversion is implemented in the view layer, keeping the data storage consistent in the database while providing a more user-friendly display in the interface.

## Index View
The `index.templ` file:
- Displays all grain weights in a responsive table
- Provides links to view, edit, and delete each grain
- Includes a button to create new grain weights
- Handles special display case for "Other" (weight = 0)

## New & Edit Views
These views provide forms for creating or updating grain weights, with:
- Input validation
- Clear labeling for the "Other" special case
- Proper CSRF protection

## Show View
The `show.templ` file displays detailed information about a specific grain weight and provides action buttons for edit and delete operations.

## Future Improvements
Possible enhancements for the future:
- Export functionality to CSV/Excel
- Filtering options for the index view
- Bulk actions for multiple grain weights
- Pagination for large datasets 