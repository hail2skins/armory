# Casings Implementation

## Overview
The Casings system provides functionality for managing ammunition casings within The Virtual Armory. Casings represent the shell or container component of ammunition rounds, with each casing type having different characteristics and popularity ratings.

## Directory Structure
The casings implementation is distributed across multiple components:

- `internal/models/casing.go`: Data model definition
- `internal/controller/admin_casing.go`: Admin controller for CRUD operations
- `internal/database/database.go`: Database service methods for casing operations
- `cmd/web/views/admin/casing/`: UI templates for the admin interface

## Data Model
The `Casing` model includes:
- `ID`: Unique identifier
- `Type`: The type/name of the casing (unique field)
- `Popularity`: Numeric rating of popularity (0-100)
- Standard GORM fields: `CreatedAt`, `UpdatedAt`, `DeletedAt` (for soft delete)

## Features

### CRUD Operations
The implementation provides complete CRUD (Create, Read, Update, Delete) functionality:
- Create: Add new casing types to the system
- Read: View individual casings or list all casings
- Update: Modify casing properties
- Delete: Soft-delete casings from the system

### Soft Delete with Restoration
A notable feature is the soft-delete restoration capability:

1. When a casing is deleted, it's not physically removed from the database but marked with a `DeletedAt` timestamp
2. If a user attempts to create a new casing with the same type as a previously deleted one:
   - The system detects the soft-deleted record
   - Restores it by clearing the `DeletedAt` timestamp
   - Updates its attributes with the new values
   - Returns the restored record with the same ID

This approach:
- Prevents duplicate key errors
- Preserves historical data and relationships
- Maintains data integrity with consistent IDs
- Provides a better user experience

## Admin Interface
The admin interface offers:
- An index page listing all casings
- Forms for creating and editing casings
- Detail view for individual casings
- Delete functionality with confirmation

## Testing
The implementation includes comprehensive test coverage:
- Unit tests for all controller methods
- Tests for the database service layer
- Specific tests for the soft-delete restoration functionality

## Usage Example

### Creating a Casing
```go
// Create a new casing
casing := &models.Casing{
    Type: "Brass",
    Popularity: 85,
}
err := dbService.CreateCasing(casing)
```

### Restoring a Deleted Casing
After a casing has been deleted:
```go
// This will find and restore the deleted casing instead of creating a new one
restoredCasing := &models.Casing{
    Type: "Brass",     // Same type as deleted casing
    Popularity: 90,    // Updated popularity
}
err := dbService.CreateCasing(restoredCasing)
```

## Future Enhancements
Potential future enhancements could include:
- Additional metadata fields like material, dimensions, etc.
- Integration with ammunition types
- Batch operations for multiple casings
- Full text search capabilities 