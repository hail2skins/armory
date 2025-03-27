# Feature Flags

## Overview
The Feature Flags system provides a way to selectively enable or disable features for specific user roles. This allows for controlled rollout of new functionality, A/B testing, and role-based access control to certain parts of the application.

## Implementation Details

### Models
- `FeatureFlag`: Represents a feature flag with name, description, and enabled status
- `FeatureFlagRole`: Represents a role assignment to a feature flag

### Controller
The `AdminFeatureFlagsController` handles all operations related to feature flags:
- `Index` - Lists all feature flags
- `Create` - Shows the form to create a new feature flag
- `Store` - Processes the creation of a new feature flag
- `Edit` - Shows the form to edit an existing feature flag
- `Update` - Processes updating an existing feature flag
- `Delete` - Removes a feature flag
- `AddRole` - Adds a role to a feature flag
- `RemoveRole` - Removes a role from a feature flag

### Views
- `index.templ` - Displays all feature flags with status and role assignments
- `create.templ` - Form for creating new feature flags
- `edit.templ` - Form for editing feature flags and managing role assignments

### Data Structures
- `FeatureFlagsViewData` - Data for the index view
- `FeatureFlagFormData` - Data for the create/edit forms
- `FeatureFlagRoleData` - Data for role assignment forms

## How It Works

1. **Feature Enabling/Disabling**:
   - A feature flag can be enabled or disabled globally
   - When disabled, the feature is unavailable to all users
   - When enabled, availability is determined by role assignments

2. **Role-Based Access**:
   - If no roles are assigned to an enabled feature flag, the feature is available to all users
   - If roles are assigned, only users with at least one of those roles can access the feature
   - Role assignments are managed through the Edit interface

3. **Integration with Application**:
   - Application code should check if a feature is available before showing UI elements or executing functionality
   - The feature service provides methods to check if a feature is available to a user

## Usage

### Creating a Feature Flag
1. Navigate to Admin > Permissions > Feature Flags
2. Click "Create New Feature Flag"
3. Enter a name (using snake_case), description, and set the initial enabled status
4. Optionally select roles that should have access to this feature
5. Click "Create Feature Flag"

### Editing a Feature Flag
1. Navigate to Admin > Permissions > Feature Flags
2. Click "Edit" for the desired feature flag
3. Modify the name, description, or enabled status as needed
4. Click "Update Feature Flag"

### Managing Role Access
1. Navigate to Admin > Permissions > Feature Flags
2. Click "Edit" for the desired feature flag
3. In the Role Management section:
   - To add a role: Select a role from the dropdown and click "Add Role"
   - To remove a role: Click the "X" button next to an existing role

### Checking Feature Availability in Code
```go
// Check if a feature is available to the current user
if featureService.IsFeatureAvailableTo(user, "feature_name") {
    // Show or enable the feature
}
```

## Best Practices

1. **Naming Conventions**:
   - Use snake_case for feature flag names (e.g., `new_dashboard`, `beta_reporting`)
   - Choose descriptive names that clearly indicate the feature's purpose
   - Avoid generic names like "test" or "feature1"

2. **Description Quality**:
   - Write clear descriptions that explain what the feature does
   - Include information about why the feature might be restricted
   - Note any dependencies or related feature flags

3. **Role Assignment**:
   - Assign the minimum necessary roles to a feature
   - Consider creating specific roles for major features
   - Remember that no role assignment means "available to all"

4. **Feature Lifecycle**:
   - Clean up old feature flags that are no longer needed
   - Consider using feature flags for all new features, even if they will be available to everyone
   - Document the expected lifecycle of temporary feature flags

5. **Testing**:
   - Test features with the flag both enabled and disabled
   - Test with different role configurations
   - Use feature flags for gradual rollouts and A/B testing

## Security Considerations
- Feature flags are not a replacement for proper authentication and authorization
- They should be used as an additional layer of control
- Always combine with proper role-based access control for sensitive features 