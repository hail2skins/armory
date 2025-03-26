# Role-Based Access Control (RBAC) System

The RBAC system provides comprehensive role-based access control for The Virtual Armory application. It replaces the previous CSV-based Casbin approach with a fully database-driven solution.

## Features

1. **Database-Backed Permissions**: All roles, permissions, and user-role assignments are stored in the database.
2. **Casbin Compatibility**: Fully compatible with Casbin's enforcement interface.
3. **Admin UI**: Complete admin UI for managing roles, permissions, and user-role assignments.
4. **Extensible Design**: Ready for integration with feature flags (coming soon).

## Technical Details

### Database Model

The RBAC system uses the `CasbinRule` model, which follows the Casbin format:

```go
type CasbinRule struct {
    ID    uint   `gorm:"primaryKey;autoIncrement"`
    Ptype string `gorm:"size:100;uniqueIndex:unique_index"`
    V0    string `gorm:"size:100;uniqueIndex:unique_index"`
    V1    string `gorm:"size:100;uniqueIndex:unique_index"`
    V2    string `gorm:"size:100;uniqueIndex:unique_index"`
    V3    string `gorm:"size:100;uniqueIndex:unique_index"`
    V4    string `gorm:"size:100;uniqueIndex:unique_index"`
    V5    string `gorm:"size:100;uniqueIndex:unique_index"`
}
```

Where:
- **Ptype**: Policy type - "p" for policy rules or "g" for role assignments
- **V0-V5**: Values for the rule:
  - For "p" rules: V0=role, V1=resource, V2=action
  - For "g" rules: V0=user, V1=role

### Key Components

1. **CasbinDBAdapter**: Custom database adapter that implements Casbin's adapter interface
2. **RBAC Helper Functions**: Functions for finding roles, users, and permissions
3. **Admin Controller**: Controller for the admin UI
4. **Admin Views**: UI for managing roles, permissions, and user-role assignments

### Role Types

The system supports three built-in roles by default:
- **admin**: Full access to all resources and actions
- **editor**: Can read, write, and update specific resources
- **viewer**: Read-only access to specific resources

### API

The RBAC system provides the following API:

```go
// Role Management
FindAllRoles(db *gorm.DB) ([]string, error)
FindPermissionsForRole(db *gorm.DB, role string) ([]map[string]string, error)
CreateRole(db *gorm.DB, role string, permissions []map[string]string) error
DeleteRole(db *gorm.DB, role string) error

// User-Role Management
FindUsersInRole(db *gorm.DB, role string) ([]string, error)
AddUserToRole(db *gorm.DB, user, role string) error
RemoveUserFromRole(db *gorm.DB, user, role string) error
IsUserInRole(db *gorm.DB, user, role string) (bool, error)
GetUserRoles(db *gorm.DB, user string) ([]string, error)

// Import/Export
ImportFromCSV(db *gorm.DB, csv string) error
ExportToCSV(db *gorm.DB) (string, error)
```

## Usage

### Admin UI

The admin UI for managing roles is available at `/admin/permissions`. From there, you can:

1. View all roles and their permissions
2. Create new roles
3. Edit existing roles
4. Delete roles
5. Assign users to roles
6. Remove users from roles
7. Import default policies

### Integration with Feature Flags (Coming Soon)

The RBAC system is designed to be integrated with feature flags. This will allow:

1. Controlled rollout of new features
2. Role-based access to features
3. A/B testing of features
4. Temporary feature enabling/disabling

## Future Enhancements

1. **Resource-Based Permissions**: Define permissions based on specific resources/entities
2. **Custom Roles**: Allow creation of custom roles with specific permissions
3. **Permission Groups**: Group permissions for easier management
4. **Audit Logging**: Track permission changes
5. **Role Hierarchies**: Create hierarchical roles (e.g., super-admin > admin > editor) 