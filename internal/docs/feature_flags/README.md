# Feature Flag System

## Overview

The feature flag system in The Virtual Armory allows controlled rollout of new features through centralized management of feature availability. It integrates with the existing Role-Based Access Control (RBAC) system to allow fine-grained control over which roles can access specific features.

## Features

- **Global Enable/Disable**: Each feature can be enabled or disabled globally
- **Role-Based Access Control**: Features can be restricted to specific roles
- **Persistent Storage**: All feature flags are stored in the database
- **Runtime Configuration**: Features can be toggled without application restart
- **Integration with RBAC**: Leverages the existing Casbin-based RBAC system

## Technical Details

### Database Model

The feature flag system uses two main models:

#### FeatureFlag
```go
type FeatureFlag struct {
    ID          uint      `gorm:"primaryKey;autoIncrement"`
    Name        string    `gorm:"size:255;uniqueIndex;not null"`
    Enabled     bool      `gorm:"default:false"`
    Description string    `gorm:"type:text"`
    CreatedAt   time.Time `gorm:"autoCreateTime"`
    UpdatedAt   time.Time `gorm:"autoUpdateTime"`
    Roles       []FeatureFlagRole `gorm:"foreignKey:FeatureFlagID"`
}
```

#### FeatureFlagRole
```go
type FeatureFlagRole struct {
    ID            uint   `gorm:"primaryKey;autoIncrement"`
    FeatureFlagID uint   `gorm:"index;not null"`
    Role          string `gorm:"size:100;not null"`
}
```

### Key Components

1. **FeatureFlagService**: Provides operations on feature flags
2. **Integration with RBAC**: Checks if a user has the required role to access a feature
3. **Feature Flag Checking**: Helper functions to check if a feature is enabled for a user

### How Feature Access Works

1. **Check if feature exists and is enabled**: A feature must exist and be enabled to be accessible
2. **Check for role restrictions**: If no roles are associated with a feature, it's available to everyone
3. **Check user roles**: If roles are associated with a feature, only users with those roles can access it

### API

The feature flag system provides the following API:

```go
// Feature Flag Management
FindAllFeatureFlags() ([]FeatureFlag, error)
FindFeatureFlagByID(id uint) (*FeatureFlag, error)
FindFeatureFlagByName(name string) (*FeatureFlag, error)
CreateFeatureFlag(flag *FeatureFlag) error
UpdateFeatureFlag(flag *FeatureFlag) error
DeleteFeatureFlag(id uint) error

// Role Management for Feature Flags
AddRoleToFeatureFlag(flagID uint, role string) error
RemoveRoleFromFeatureFlag(flagID uint, role string) error

// Feature Access Checking
IsFeatureEnabled(featureName string) (bool, error)
CanAccessFeature(db *gorm.DB, username, featureName string) (bool, error)
```

## Usage

### Creating a Feature Flag

```go
service := models.NewFeatureFlagService(db)

flag := &models.FeatureFlag{
    Name:        "new_ammo_feature",
    Enabled:     false,
    Description: "New ammunition tracking feature",
}

err := service.CreateFeatureFlag(flag)
```

### Restricting a Feature to Roles

```go
// Add role restriction to the feature
err := service.AddRoleToFeatureFlag(flagID, "admin")
err = service.AddRoleToFeatureFlag(flagID, "editor")
```

### Checking Feature Access in Controllers

```go
// In a controller function
func (c *Controller) SomeFeatureHandler(ctx *gin.Context) {
    username := ctx.GetString("username")
    
    // Check if user can access the feature
    canAccess, err := models.CanAccessFeature(c.db, username, "new_ammo_feature")
    if err != nil {
        // Handle error
        ctx.AbortWithStatus(http.StatusInternalServerError)
        return
    }
    
    if !canAccess {
        // Feature is not available to this user
        ctx.AbortWithStatus(http.StatusForbidden)
        return
    }
    
    // Continue with the feature implementation
    // ...
}
```

### Checking Feature Access in Templates

You can pass feature access information to templates via the view data:

```go
// In controller
featureAccess := make(map[string]bool)
canAccess, _ := models.CanAccessFeature(c.db, username, "new_ammo_feature")
featureAccess["new_ammo_feature"] = canAccess

viewData := data.ViewData{
    // ... other data
    Data: &data.SomeViewData{
        // ... view-specific data
        FeatureAccess: featureAccess,
    },
}

// In template
{{ if .Data.FeatureAccess.new_ammo_feature }}
    <!-- Show feature UI -->
{{ end }}
```

## Integration with Existing Features

When implementing a new feature that should be behind a feature flag:

1. Create a feature flag for the new feature
2. Restrict access to appropriate roles if needed
3. Check for feature access before serving API endpoints or rendering UI elements

## Future Enhancements

1. **Time-based activation**: Enable features at a specific date/time
2. **Progressive rollout**: Enable features for a percentage of users
3. **A/B testing**: Show different versions of a feature to different users
4. **User-specific flags**: Enable features for specific users regardless of roles
5. **Admin UI**: Web interface for managing feature flags 