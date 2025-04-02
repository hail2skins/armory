# Admin Munitions Management

## Overview
The Admin Munitions Management feature provides administrators with the ability to view and manage all ammunition tracked in The Virtual Armory. This feature complements the existing Gun Management system and allows administrators to monitor ammunition data across all users.

## Features
- View a centralized list of all ammunition tracked in the system
- Filter and search ammunition by any field (owner, brand, caliber, etc.)
- Real-time round count that updates based on search/filter results
- Detailed view of individual ammunition items
- Integration with the admin dashboard and sidebar

## Pages
1. **Index Page** (`/admin/munitions`): Displays a table of all ammunition with key information including owner, name, brand, caliber, bullet style, grain, casing, count, and acquisition date.
2. **Show Page** (`/admin/munitions/:id`): Shows detailed information about a specific ammunition item, including cost per round calculations and time owned statistics.

## Technical Implementation
- **Controller**: `AdminMunitionsController` in `internal/controller/admin_munitions.go`
- **Templates**: Located in `cmd/web/views/admin/munition/`
  - `index.templ`: The main ammunition list view
  - `show.templ`: Detailed view for individual ammunition
- **Data Structures**:
  - `MunitionsIndexData`: Data structure for the index page
  - `MunitionShowData`: Data structure for the show page

## Database Interactions
The controller interacts with the database through the following methods:
- `FindAllAmmo()`: Retrieves all ammunition records
- `FindAmmoByID(id)`: Retrieves a specific ammunition record
- `CountAmmoByUser(userID)`: Counts ammunition records for a specific user

## UI Features
- **Sorting**: Users can sort any column by clicking the column header
- **Searching**: Real-time search functionality filters ammunition by any attribute
- **Total Rounds Counter**: Updates dynamically based on filtered results

## Security
- Access to ammunition management is controlled through Casbin roles and permissions
- Routes are secured behind authentication and authorization checks
- Users must have the appropriate permissions to access these pages

## Future Enhancements
- Add ability to edit ammunition details
- Enable direct management of ammunition counts (add/remove rounds)
- Add batch operations for multiple ammunition records
- Implement statistical reporting (usage over time, cost analysis) 