# Bullet Style Model

## Overview
The Bullet Style model represents different projectile types used in ammunition within The Virtual Armory application. Examples include FMJ (Full Metal Jacket), JHP (Jacketed Hollow Point), and SP (Soft Point).

## Model Definition

The BulletStyle model is defined in `internal/models/bullet_style.go`:

```go
type BulletStyle struct {
	gorm.Model
	Type       string `gorm:"type:varchar(100);not null;uniqueIndex" json:"type"`
	Nickname   string `gorm:"type:varchar(100)" json:"nickname"`
	Popularity int    `gorm:"default:0" json:"popularity"`
}
```

## Fields

| Field | Type | Description | Database Constraints |
|-------|------|-------------|---------------------|
| ID | uint | Primary key | Auto-increment |
| Type | string | Primary identifier of the bullet style | varchar(100), not null, unique index |
| Nickname | string | Alternative name or description | varchar(100), optional |
| Popularity | int | Sorting weight (higher values appear first) | default:0 |
| CreatedAt | time.Time | Creation timestamp | Provided by gorm.Model |
| UpdatedAt | time.Time | Last update timestamp | Provided by gorm.Model |
| DeletedAt | gorm.DeletedAt | Soft deletion timestamp | Provided by gorm.Model, nullable |

## Database Operations

The database operations for the BulletStyle model are defined in the database service interface:

```go
// From internal/database/database.go
type Service interface {
    // ... other methods ...
    
    // Bullet Style methods
    FindAllBulletStyles() ([]models.BulletStyle, error)
    FindBulletStyleByID(id uint) (*models.BulletStyle, error)
    CreateBulletStyle(bulletStyle *models.BulletStyle) error
    UpdateBulletStyle(bulletStyle *models.BulletStyle) error
    DeleteBulletStyle(id uint) error
}
```

These methods are implemented in the database service implementation.

## Usage Example

```go
// Creating a new bullet style
bulletStyle := models.BulletStyle{
    Type:       "FMJ",
    Nickname:   "Full Metal Jacket",
    Popularity: 100,
}
err := dbService.CreateBulletStyle(&bulletStyle)

// Finding all bullet styles
bulletStyles, err := dbService.FindAllBulletStyles()

// Finding a bullet style by ID
bulletStyle, err := dbService.FindBulletStyleByID(1)

// Updating a bullet style
bulletStyle.Nickname = "Updated Nickname"
err := dbService.UpdateBulletStyle(bulletStyle)

// Deleting a bullet style
err := dbService.DeleteBulletStyle(1)
```

## Data Seeding

Initial data for bullet styles is provided by the seed function in `internal/database/seed/bullet_styles.go`:

```go
func SeedBulletStyles(db *gorm.DB) {
    bulletStyles := []models.BulletStyle{
        {Type: "Other", Popularity: 999},
        {Type: "FMJ", Nickname: "Full Metal Jacket", Popularity: 100},
        {Type: "JHP", Nickname: "Jacketed Hollow Point", Popularity: 90},
        // Additional predefined bullet styles...
    }
    
    // Implementation to seed the database...
}
```

This ensures that common bullet styles are available when the application is first run.

## Relationships

The BulletStyle model can be related to other models in the application, such as:

- Ammunition types
- Reloading data
- Ballistic information

These relationships can be implemented as needed for specific application features.

## Validation

Validation for bullet styles includes:

- Type is required and must be unique
- Nickname is optional
- Popularity must be a non-negative integer

## Testing

The BulletStyle model and its database operations are tested in:

- `internal/models/bullet_style_test.go` (model tests)
- `internal/database/bullet_style_test.go` (database operations tests)

Tests cover CRUD operations, validation rules, and edge cases.

## UI Integration

The BulletStyle model is used in the admin area for bullet style management and will be integrated into various ammunition-related features of the application. 