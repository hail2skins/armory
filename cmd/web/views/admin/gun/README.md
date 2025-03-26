# Admin Guns Tracking Feature

## Overview
The Admin Guns Tracking feature allows administrators to view and analyze all guns being tracked within the system. This feature provides a comprehensive dashboard that displays all guns along with their details, and includes filtering, searching, and sorting capabilities.

## Features

### Guns Tracking Dashboard
- **Total Guns Counter**: Dynamically displays the total number of guns being tracked in the system
- **Filtering**: Updates the counter based on filtered results
- **Searchable Table**: Users can search for guns by any field (owner, name, purpose, etc.)
- **Sortable Columns**: Click on column headers to sort the table by that column
- **Full Details**: Displays comprehensive information about each gun, including:
  - Owner's name
  - Gun name
  - Serial number
  - Purpose
  - Weapon type
  - Caliber
  - Manufacturer
  - Acquisition date

## Implementation Details

### Components
- `cmd/web/views/admin/gun/index.templ`: Template for the admin guns index page
- `internal/controller/admin_guns.go`: Controller handling admin gun routes
- `internal/database/database.go`: Extended with gun-related query methods
- `tests/admin_guns_routes_test.go`: Tests for the admin guns routes

### Database Methods
- `FindAllGuns()`: Retrieves all guns from the database
- `FindAllUsers()`: Retrieves all users
- `CountGunsByUser()`: Counts guns owned by a specific user
- `FindAllCalibersByIDs()`: Gets calibers by their IDs
- `FindAllWeaponTypesByIDs()`: Gets weapon types by their IDs

### JavaScript Functionality
- `searchTable()`: Filters the table based on user search input and updates the counter
- `sortTable()`: Sorts the table by the selected column

## Admin Dashboard Integration
The Admin Dashboard page includes a "Guns" column that displays the count of guns for each user. This count is a clickable link to the guns page filtered for that specific user.

## Sidebar Integration
The feature is accessible from the admin sidebar under the "User Management" section, with a dedicated gun icon.

## Future Enhancements
- Export gun data to CSV/Excel
- Implement advanced filtering (by date range, caliber, weapon type, etc.)
- Add statistics and visualizations for gun ownership trends
- Implement admin actions like approving or flagging certain guns

## Security Considerations
This feature is restricted to admin users only and respects the site's RBAC (Role-Based Access Control) through the Casbin integration. 