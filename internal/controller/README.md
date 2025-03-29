# Controllers Documentation

## Overview
The controllers in The Virtual Armory handle the application logic, processing requests from users, interacting with the database through the service layer, and returning appropriate responses. They implement the C (Controller) part of the MVC architecture.

## Controller Structure
Each controller is organized around a specific domain entity and follows similar patterns:

1. Controller struct definition with required dependencies
2. Constructor function
3. Handler methods for HTTP endpoints
4. Helper functions specific to that controller

## Admin Casing Controller

### Overview
The `AdminCasingController` manages all operations related to ammunition casings in the admin interface. It provides CRUD functionality for casings with proper validation, error handling, and success messaging.

### File Location
`internal/controller/admin_casing.go`

### Methods

#### Constructor
```go
func NewAdminCasingController(db database.Service) *AdminCasingController
```
Creates a new controller instance with the database service dependency.

#### Index
```go
func (c *AdminCasingController) Index(ctx *gin.Context)
```
Lists all casings with pagination and sorting options.

#### New
```go
func (c *AdminCasingController) New(ctx *gin.Context)
```
Displays the form for creating a new casing.

#### Create
```go
func (c *AdminCasingController) Create(ctx *gin.Context)
```
Processes the form submission to create a new casing. Includes special handling for soft-deleted casings with the same type.

#### Show
```go
func (c *AdminCasingController) Show(ctx *gin.Context)
```
Displays detailed information about a specific casing.

#### Edit
```go
func (c *AdminCasingController) Edit(ctx *gin.Context)
```
Displays the form for editing an existing casing.

#### Update
```go
func (c *AdminCasingController) Update(ctx *gin.Context)
```
Processes the form submission to update an existing casing.

#### Delete
```go
func (c *AdminCasingController) Delete(ctx *gin.Context)
```
Processes the request to soft-delete a casing.

### Key Features

#### Soft Delete Restoration
When a user attempts to create a casing with the same type as a previously deleted one, the controller:
1. Passes the request to the database service
2. The service detects the soft-deleted record and restores it
3. The controller handles the response appropriately, redirecting with a success message

#### Context Data Handling
The controller uses a helper function `getAdminCasingDataFromContext` to:
1. Extract admin data from the Gin context
2. Set appropriate titles and paths
3. Add success/error messages
4. Package all necessary data for the views

#### Form Validation
Input validation includes:
- Required field checks
- Data type validation
- Error reporting back to the user

#### Flash Messaging
The controller sets flash messages for:
- Successful operations
- Validation errors
- Database errors

### Integration Points

#### Routes Registration
Routes are registered in `internal/server/admin_routes.go`:
```go
casingGroup := adminGroup.Group("/casings")
{
    casingGroup.GET("", adminCasingController.Index)
    casingGroup.GET("/new", adminCasingController.New)
    casingGroup.POST("", adminCasingController.Create)
    casingGroup.GET("/:id", adminCasingController.Show)
    casingGroup.GET("/:id/edit", adminCasingController.Edit)
    casingGroup.POST("/:id", adminCasingController.Update)
    casingGroup.POST("/:id/delete", adminCasingController.Delete)
}
```

#### Authentication & Authorization
The controller relies on:
- Authentication middleware for user verification
- Casbin for role-based access control
- CSRF protection for form submissions

## Testing
The controller includes comprehensive test coverage in `internal/controller/admin_casing_test.go`:
- Tests for all CRUD operations
- Validation testing
- Error handling tests
- Soft-delete restoration test

## Future Enhancements
Potential improvements include:
- Batch operations for multiple casings
- Advanced filtering and search
- Audit logging for admin actions
- Additional validation rules 