# Virtual Armory Login "Hiccup" Investigation Report

**Investigated by:** Agetha (Nagetha)  
**Date:** 2026-02-12  
**Branch:** dev  

---

## EXECUTIVE SUMMARY

I investigated the reported "hiccup" (delay/stutter) during user login to The Virtual Armory. Art observed that TWO different database calls occur during the login flow.

## FINDINGS

### 1. THE LOGIN FLOW DATABASE CALLS

During the login flow, there are indeed **TWO database queries** that happen:

**Request 1: POST /login**
```
LoginHandler → AuthenticateUser() → GetUserByEmail() [DB CALL #1]
```

**Request 2: GET /owner (redirect after login)**
```
LandingPage → GetUserByEmail() [DB CALL #2]
```

This is because:
1. Login authenticates the user (queries DB for auth)
2. Login stores minimal auth info (user ID, email) in session/cache
3. Redirect to /owner is a NEW HTTP request
4. Owner page fetches full user record again from DB

### 2. UNUSED AUTHENTICATION STRATEGY (DEAD CODE)

In `NewAuthController()`, there's code that sets up a go-guardian authentication strategy:

```go
strategy := basic.NewCached(func(ctx context.Context, r *http.Request, username, password string) (auth.Info, error) {
    // ... calls db.AuthenticateUser()
}, cache)
```

**This strategy is NEVER USED.** The `strategy` field is stored in the AuthController but:
- Never called by LoginHandler
- Never used in any middleware
- Never referenced anywhere in the codebase

This appears to be dead code from an incomplete OAuth/API authentication implementation.

### 3. CODE CLEANUP OPPORTUNITY

In the LoginHandler, there were separate error checks:
```go
user, err := a.db.AuthenticateUser(...)
if err != nil {
    // handle error
}
if user == nil {
    // handle nil user
}
```

Since `AuthenticateUser` returns `(nil, nil)` on authentication failure (not error), these checks were redundant.

---

## FIXES APPLIED

### Fix 1: Consolidated Error Handling in LoginHandler
**File:** `internal/controller/auth.go`

Consolidated the redundant error checks into one:
```go
user, err := a.db.AuthenticateUser(c.Request.Context(), email, password)
if err != nil || user == nil {
    // Handle all auth failures in one place
}
```

### Fix 2: Optimized Subscription Check Updates
**File:** `internal/database/user_service.go`

Changed `CheckExpiredPromotionSubscription()` to use `Updates()` instead of `Save()`:
```go
// Before: s.db.Save(user) - updates ALL fields
// After: s.db.Model(user).Updates({...}) - updates only changed fields
```

This reduces database write overhead by only updating the specific fields that changed.

---

## ANALYSIS: IS THIS REALLY A "HICCUP"?

The two database calls are actually **correct behavior** for a multi-request flow:

1. **First call** authenticates credentials and updates login metadata
2. **Second call** loads user data for the landing page

These are separate HTTP requests, each needing fresh data. The "hiccup" Art experiences is likely the cumulative latency of:
- Login request (DB query + bcrypt password verification)
- Redirect overhead
- Owner page request (another DB query)

### Potential Further Optimizations

1. **Cache user data across requests:** Store full user object in Redis/session
2. **Eliminate unused strategy code:** Remove the go-guardian strategy setup
3. **Optimize owner page queries:** Use JOINs or eager loading where possible

---

## VERIFICATION

All changes verified with:
```bash
go test ./internal/database/...     # PASS
go test ./internal/controller/...   # PASS
go build ./...                      # SUCCESS
```

---

## RECOMMENDATIONS

1. **Consider removing the unused go-guardian strategy** - it's dead code that adds startup overhead
2. **Monitor query performance** - The two queries shouldn't cause noticeable hiccups on their own
3. **Check for N+1 queries** in the owner pages - there are many GetUserByEmail calls throughout

---

## FILES MODIFIED

1. `internal/controller/auth.go` - Consolidated error handling
2. `internal/database/user_service.go` - Optimized subscription check updates

**Branch:** dev (NOT main - as requested)
