# Ammunition Management

This feature allows users to manage their ammunition inventory in The Virtual Armory application.

## Overview

The ammunition management system allows users to:

1. Add new ammunition to their inventory
2. Track ammunition count
3. Record acquisition dates and costs
4. Associate ammunition with brands, calibers, bullet styles, grain weights, and casing types
5. Search and filter ammunition by various attributes

## Technical Implementation

### Models

- `Ammo` - Core ammunition model that stores all ammunition data
- Related models: `Brand`, `Caliber`, `BulletStyle`, `Grain`, and `Casing`

### Controllers

- `OwnerController.AmmoNew` - Displays the form for adding new ammunition
- `OwnerController.AmmoCreate` - Processes form submission and creates new ammunition entries
- (Future) `OwnerController.AmmoIndex` - Lists all ammunition
- (Future) `OwnerController.AmmoEdit` - Edits existing ammunition
- (Future) `OwnerController.AmmoUpdate` - Updates ammunition after edits
- (Future) `OwnerController.AmmoDelete` - Deletes ammunition entries

### Views

- `munitions/new.templ` - Form for adding new ammunition
- (Future) `munitions/index.templ` - List of ammunition
- (Future) `munitions/edit.templ` - Form for editing ammunition
- (Future) `munitions/show.templ` - Detailed view of ammunition

### Validation

Ammunition validation includes:
- Required fields: name, count, brand, and caliber
- Name length limited to 100 characters
- Count must be positive
- Price must be non-negative
- Acquisition date cannot be in the future
- Foreign key integrity for brands, calibers, bullet styles, grains, and casings

### Searchable Dropdowns

The form implements searchable dropdowns for:
- Brands
- Calibers
- Bullet Styles
- Grain Weights
- Casing Types

These dropdowns are ordered by popularity first, then alphabetically to provide a better user experience.

## Permissions

Access to ammunition management features is restricted by role-based permissions using Casbin. Users must have appropriate roles to access the ammunition management features.

## Testing

### CSRF Protection

When writing tests for ammunition management features (and other CSRF-protected routes), you must handle CSRF tokens appropriately. There are three ways to bypass CSRF validation in tests:

1. **Enable Test Mode** (Recommended):
   ```go
   // In your test suite's SetupSuite method
   middleware.EnableTestMode()
   
   // In your test suite's TearDownSuite method
   middleware.DisableTestMode()
   ```

2. **Set Environment Variable**:
   ```go
   os.Setenv("GO_ENV", "test")
   // Run tests
   os.Unsetenv("GO_ENV") // Remember to unset after tests
   ```

3. **Add CSRF Bypass Header**:
   ```go
   req.Header.Set("X-Test-CSRF-Bypass", "true")
   ```

Additionally, for forms that require a CSRF token, you need to either:
- Add a CSRF token to form data: `"csrf_token": {"test-csrf-token"}`
- Add a CSRF token to the context: `c.Set("csrf_token", "test-csrf-token")`

### Example Test

See `tests/owner_ammo_controller_test.go` for examples of testing ammunition management features.

## Future Enhancements

1. Ammunition inventory tracking (usage log)
2. Batch operations for adding multiple ammunition items
3. Import/export functionality
4. Ammunition statistics dashboard
5. Alerts for low ammunition counts 