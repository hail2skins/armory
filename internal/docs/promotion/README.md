# Promotion Feature

## Overview

The promotion feature in The Virtual Armory allows administrators to create and manage time-limited promotional offers for users. Promotions can be applied automatically during registration or login, providing subscription benefits for a specified number of days.

## Features

- **Time-limited promotions**: Set start and end dates for each promotion
- **Different promotion types**: Support for "free_trial", "discount", and "special_offer" types
- **Benefit days**: Specify how many days of benefits users receive
- **Home page display**: Option to feature promotions on the home page
- **Apply to new users**: Automatically apply benefits to newly registered users
- **Apply to existing users**: Optionally apply benefits to existing users when they log in
- **Adaptive UI**: Banner text and buttons change based on whether a promotion applies to new users only or to both new and existing users

## How It Works

### Promotion Management

Administrators can manage promotions through the admin interface at `/admin/promotions`. The following operations are supported:

- **Create** new promotions
- **View** promotion details
- **Edit** existing promotions
- **Delete** promotions

### Promotion Application

Promotions are applied automatically under two scenarios:

1. **New User Registration**: When a new user registers, if there are active promotions, the system will apply the best available promotion (the one with the most benefit days, or if tied, the one ending soonest).

2. **Existing User Login**: If a promotion has the "Apply to Existing Users" option enabled, it will be applied when an existing user logs in (if they don't already have an active better subscription).

### Promotion Banner UI

The promotion banner displayed on the site adapts based on the promotion settings:

- For promotions that apply only to new users:
  - Banner shows "Register now and get X days free!"
  - Only a "Sign Up" button is displayed

- For promotions that apply to both new and existing users:
  - Banner shows "Register OR Login and get X days free!"
  - Both "Sign Up" and "Login" buttons are displayed

This adaptive UI helps clearly communicate to users how they can take advantage of the active promotion.

### Promotion Model

The promotion model includes the following fields:

| Field | Type | Description |
|-------|------|-------------|
| Name | string | The name of the promotion |
| Type | string | The type of promotion ("free_trial", "discount", "special_offer") |
| Active | bool | Whether the promotion is currently active |
| StartDate | time.Time | When the promotion starts |
| EndDate | time.Time | When the promotion ends |
| BenefitDays | int | Number of days the promotion benefits last |
| DisplayOnHome | bool | Whether to display on the home page |
| ApplyToExistingUsers | bool | Whether to apply to existing users when they log in |
| Description | string | Marketing description for the promotion |
| Banner | string | Path to banner image for promotion display |

## Usage Examples

### Creating a New User Promotion

1. Navigate to `/admin/promotions/new`
2. Fill in the promotion details:
   - Name: "Spring Free Trial"
   - Type: "free_trial"
   - Active: Yes
   - Start/End Dates: Set appropriate dates
   - Benefit Days: 30
   - Display on Home: Yes
   - Apply to Existing Users: No (to only apply to new registrations)
3. Save the promotion

### Creating a Promotion for All Users

1. Navigate to `/admin/promotions/new`
2. Fill in the promotion details as above
3. Set "Apply to Existing Users" to Yes
4. Save the promotion

When this promotion is active:
- New users will receive benefits on registration
- Existing users will receive benefits on login

## Technical Details

### How Promotions Are Applied

When applying a promotion to a user, the system:

1. Updates the user's subscription tier to "promotion"
2. Sets the subscription status to "active"
3. Calculates the subscription end date based on the current date plus the promotion's benefit days
4. Records the promotion ID on the user record

### Finding the Best Active Promotion

The system selects the best active promotion using this logic:

1. Filter for active promotions where the current date is between start and end dates
2. If multiple active promotions exist, choose the one with the most benefit days
3. If tied on benefit days, choose the one ending soonest (to create urgency)

## Admin UI

The promotion management interface includes:

- **Index page**: Lists all promotions with key information and actions
- **Show page**: Displays full details of a single promotion
- **New/Edit forms**: Forms for creating and editing promotions

## Integration with Auth Flow

The promotion system is integrated with the authentication flow:

- The `RegisterHandler` checks for active promotions and applies them to new users
- The `LoginHandler` checks for active promotions with "Apply to Existing Users" enabled and applies them to existing users

## Future Enhancements

Potential future enhancements to the promotion system:

- Promo codes for manual redemption
- Targeted promotions for specific user segments
- More sophisticated promotion types (percentage discounts, tiered benefits)
- Promotion analytics to track effectiveness 