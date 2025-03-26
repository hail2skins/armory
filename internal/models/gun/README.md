# Gun Model

The Gun model represents a firearm in The Virtual Armory system. This document outlines the fields, validation rules, and functionality related to guns in the application.

## Fields

| Field           | Type           | Description                                        | Validation                            |
|-----------------|----------------|----------------------------------------------------|---------------------------------------|
| Name            | string         | The name of the firearm                            | Required, max 100 characters          |
| SerialNumber    | string         | The serial number of the firearm                   | Optional                              |
| Purpose         | string         | The purpose of the gun (e.g., "Carry", "Home Defense") | Optional, max 100 characters     |
| Acquired        | *time.Time     | The date the firearm was acquired                  | Optional, cannot be in the future     |
| WeaponTypeID    | uint           | ID of the weapon type                              | Required, must exist in database      |
| WeaponType      | WeaponType     | Relationship to the weapon type                    | -                                     |
| CaliberID       | uint           | ID of the caliber                                  | Required, must exist in database      |
| Caliber         | Caliber        | Relationship to the caliber                        | -                                     |
| ManufacturerID  | uint           | ID of the manufacturer                             | Required, must exist in database      |
| Manufacturer    | Manufacturer   | Relationship to the manufacturer                   | -                                     |
| OwnerID         | uint           | ID of the user who owns the gun                    | Required                              |
| Paid            | *float64       | The price paid for the gun in USD                 | Optional, cannot be negative          |

## Validation Rules

The Gun model has several validation rules:

1. **Name**: Cannot exceed 100 characters
2. **Purpose**: Cannot exceed 100 characters (optional field)
3. **Acquired Date**: Cannot be in the future
4. **Paid Amount**: Cannot be negative
5. **Foreign Keys**: WeaponTypeID, CaliberID, and ManufacturerID must exist in their respective tables

## CRUD Operations

The following methods are available for Gun management:

- `FindGunsByOwner(db, ownerID)`: Retrieves all guns belonging to a specific owner
- `FindGunByID(db, id, ownerID)`: Retrieves a specific gun by ID, ensuring it belongs to the specified owner
- `CreateGunWithValidation(db, gun)`: Creates a new gun with validation
- `UpdateGunWithValidation(db, gun)`: Updates an existing gun with validation
- `DeleteGun(db, id, ownerID)`: Deletes a gun, ensuring it belongs to the specified owner

## Subscription Tier Limits

Free tier users have access limitations:
- Only the first 2 guns are accessible in the dashboard view
- A subscription upgrade prompt appears when this limit is reached

## User Interface

The Gun model is used in several views:
- Owner dashboard: Displays a summarized table of the user's guns
- Arsenal page: Provides a detailed view with sorting and searching
- Gun detail page: Shows all information about a specific gun
- Add/Edit forms: Allow users to create and modify gun entries

## Purpose Field

The Purpose field was added to allow users to specify the intended purpose of their firearm. Examples include:
- Home Defense
- Carry
- Plinking
- Target Shooting
- Hunting
- Competition
- Collection

This field helps users categorize their firearms based on function rather than just specifications. The Purpose field is:
- Optional
- Limited to 100 characters
- Displayed in the gun details view
- Editable through the gun forms
- Not displayed in the main owner dashboard table

## Recent Changes

The Purpose field was recently added to the Gun model to enhance categorization capabilities. The field is fully integrated with the existing validation system and CRUD operations. 