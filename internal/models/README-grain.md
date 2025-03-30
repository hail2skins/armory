# Grain Model

## Overview
The Grain model represents different bullet weights measured in grains within The Virtual Armory application. Bullet weight is a critical factor in ammunition ballistics, affecting velocity, trajectory, energy delivery, and recoil.

## Model Definition

The Grain model is defined in `internal/models/grain.go`:

```go
type Grain struct {
	gorm.Model
	Weight     int `gorm:"unique;not null"` // Grain weight (e.g., 115, 55)
	Popularity int `gorm:"not null;default:0"`
}
```

## Fields

| Field | Type | Description | Database Constraints |
|-------|------|-------------|---------------------|
| ID | uint | Primary key | Auto-increment |
| Weight | int | Weight of the bullet in grains | unique, not null |
| Popularity | int | Sorting weight (higher values appear first) | not null, default:0 |
| CreatedAt | time.Time | Creation timestamp | Provided by gorm.Model |
| UpdatedAt | time.Time | Last update timestamp | Provided by gorm.Model |
| DeletedAt | gorm.DeletedAt | Soft deletion timestamp | Provided by gorm.Model, nullable |

## Database Operations

The model includes the following functions for interacting with the database:

- `FindAllGrains(db *gorm.DB) ([]Grain, error)` - Retrieves all grain weights, ordered by popularity and weight
- `FindGrainByID(db *gorm.DB, id uint) (*Grain, error)` - Retrieves a specific grain weight by ID
- `FindGrainByWeight(db *gorm.DB, weight int) (*Grain, error)` - Retrieves a specific grain weight by its weight value
- `CreateGrain(db *gorm.DB, grain *Grain) error` - Creates a new grain weight record
- `UpdateGrain(db *gorm.DB, grain *Grain) error` - Updates an existing grain weight record
- `DeleteGrain(db *gorm.DB, id uint) error` - Soft-deletes a grain weight record

## Common Grain Weights

The application seeds common grain weights for different calibers:

- Pistol calibers: 115gr, 124gr, 147gr (9mm); 180gr, 165gr (.40 S&W); 230gr, 185gr (.45 ACP)
- Rifle calibers: 55gr, 62gr, 77gr (5.56/.223); 123gr (7.62x39); 150gr, 168gr, 175gr (.308/7.62 NATO)
- Rimfire: 36gr, 40gr (.22 LR)
- Other examples: 158gr (.38 Special/.357 Magnum), 240gr (.44 Magnum), etc.

## Significance in Ammunition

Grain weight affects:
- Velocity: Lighter bullets typically travel faster
- Energy: Heavier bullets retain energy better at distance
- Recoil: Heavier bullets often produce more felt recoil
- Purpose: Different weights are optimal for different applications (target shooting, hunting, defense)

## Usage Example

```go
// Creating a new grain weight
grain := &models.Grain{
    Weight:     150,
    Popularity: 85,
}
err := models.CreateGrain(db, grain)

// Finding all grain weights
grains, err := models.FindAllGrains(db)

// Finding by weight
grain, err := models.FindGrainByWeight(db, 150)
``` 