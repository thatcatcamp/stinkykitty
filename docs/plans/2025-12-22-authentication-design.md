# Authentication System Design

**Date:** 2025-12-22
**Status:** Approved for Implementation

## Overview

JWT-based authentication system for StinkyKitty admin panel. Uses HTTP-only cookies for browser-only access with simple site-level permissions and global admin support.

## Design Decisions

### Session Management
**Choice:** JWT tokens (stateless)
- No server-side session storage needed
- Scales well across multiple servers
- 8-hour token expiry balances security and convenience
- Tokens stored in HTTP-only cookies (XSS protection)

**Rejected Alternatives:**
- Server-side sessions: Adds complexity, requires Redis/storage
- Hybrid JWT + refresh tokens: Overkill for admin panel use case

### Token Storage
**Choice:** HTTP-only cookie
- Automatic inclusion in requests
- Protected from XSS (JavaScript can't access)
- SameSite=Lax provides CSRF protection
- Perfect for browser-only admin panel

**Rejected Alternatives:**
- localStorage + Authorization header: Vulnerable to XSS
- Cookie + CSRF token: Unnecessary complexity

### Permissions Model
**Choice:** Simple site membership + global admin flag
- Users are site members (full access) or not
- `User.IsGlobalAdmin` boolean for platform administrators
- Dead simple, perfect for small trusted teams

**Rejected Alternatives:**
- Per-site roles (Owner/Editor/Viewer): YAGNI for now
- Complex RBAC: Major overkill

## Architecture

### Login Flow

1. User visits `/admin/login` and submits email + password
2. Server validates credentials against `User.PasswordHash` (bcrypt)
3. Server checks site access:
   - User is site owner (`Site.OwnerID` matches), OR
   - User is in `SiteUsers` table for this site, OR
   - User has `IsGlobalAdmin` flag set
4. If valid, server generates JWT containing:
   - `user_id`: User's database ID
   - `email`: User's email
   - `site_id`: Current site ID
   - `is_global_admin`: Boolean flag
   - `exp`: Expiry timestamp (8 hours from now)
5. Server sets HTTP-only cookie named `stinky_token`:
   - `HttpOnly: true` (prevents JavaScript access)
   - `Secure: true` (HTTPS only in production)
   - `SameSite: Lax` (CSRF protection)
   - `Max-Age: 28800` (8 hours in seconds)
6. Server redirects to `/admin/dashboard`

### Authentication Middleware

Runs on all `/admin/*` routes except `/admin/login`:
1. Reads `stinky_token` cookie
2. Validates JWT signature and expiry
3. Loads user from database
4. Checks site access permission
5. Sets user info in Gin context for handlers
6. Returns 401 if invalid/expired, redirects browser to login

### Middleware Chain

Updated chain for `/admin/*` routes:
```
1. Gin Logger
2. Gin Recovery
3. Site Resolution
4. IP Filter
5. Rate Limiter (login endpoint only)
6. Auth Middleware (NEW - all routes except /login)
7. Handler
```

## Database Schema

### User Model Changes

Add to `models.User`:
```go
IsGlobalAdmin bool `gorm:"default:false"`
```

**No changes needed to:**
- `Site` model (existing `OwnerID` relationship works)
- `SiteUser` model (existing many-to-many works)

## Configuration

Add to `config.yaml`:
```yaml
auth:
  jwt_secret: "CHANGE_ME_IN_PRODUCTION"  # Used to sign JWT tokens
  jwt_expiry_hours: 8                    # Token lifetime
  bcrypt_cost: 12                        # Password hashing cost
```

**Environment Variable Override:**
- `STINKY_JWT_SECRET` takes precedence over config file
- Must be random, 32+ characters, never committed to git

## Implementation Components

### New Package: `internal/auth/`

**1. `jwt.go`** - JWT token operations
```go
type Claims struct {
    UserID        uint   `json:"user_id"`
    Email         string `json:"email"`
    SiteID        uint   `json:"site_id"`
    IsGlobalAdmin bool   `json:"is_global_admin"`
    jwt.RegisteredClaims
}

func GenerateToken(user *models.User, site *models.Site) (string, error)
func ValidateToken(tokenString string) (*Claims, error)
func RefreshToken(claims *Claims) (string, error) // Future use
```

**2. `middleware.go`** - Authentication middleware
```go
func RequireAuth() gin.HandlerFunc
func RequireGlobalAdmin() gin.HandlerFunc
```

**3. `password.go`** - Password utilities
```go
func HashPassword(password string) (string, error)  // bcrypt hash
func CheckPassword(password, hash string) bool      // verify password
```

### Handler Updates

**Update `cmd/stinky/server.go`:**
- Replace placeholder `/admin/login` with actual handler
- Add `/admin/logout` endpoint (clears cookie)
- Add auth middleware to admin routes

**Create `internal/handlers/admin.go`:**
```go
func LoginHandler(c *gin.Context)      // POST email/password, validate, set cookie
func LogoutHandler(c *gin.Context)     // Clear cookie, redirect
func DashboardHandler(c *gin.Context)  // Show user info, replace placeholder
```

## Error Handling

### Login Errors

**Invalid credentials:**
- Return 401 with message: "Invalid email or password"
- Same message for both wrong email and wrong password (prevent enumeration)
- Rate limiter prevents brute force (5 attempts/minute)

**No site access:**
- Return 403 with message: "You don't have access to this site"
- User exists and password correct, but not authorized for this site

### Auth Middleware Errors

**No token / Invalid token / Expired token:**
- Return 401 Unauthorized
- Redirect browser requests to `/admin/login?redirect=/admin/dashboard`
- Future API requests return JSON error

**User lost site access:**
- Return 403 Forbidden
- Message: "Your access to this site has been revoked"
- Happens if user was removed from site while logged in

**User deleted:**
- Return 401 Unauthorized
- Redirect to login

## Security Measures

1. **Constant-time password comparison** - Prevents timing attacks
2. **No email enumeration** - Same error message for invalid email or password
3. **Rate limiting** - Already implemented (5 login attempts/minute per IP)
4. **HTTP-only cookies** - Prevents XSS token theft
5. **SameSite cookies** - Prevents CSRF attacks
6. **Secure flag** - HTTPS-only cookies in production
7. **Bcrypt cost 12** - Strong password hashing (~250ms)
8. **JWT signature validation** - Prevents token tampering
9. **Token expiry** - 8-hour lifetime limits exposure
10. **Failed login logging** - Track suspicious activity

## Testing Strategy

### Unit Tests

**`internal/auth/jwt_test.go`:**
- Test token generation with valid user/site
- Test token validation with valid token
- Test token validation with expired token
- Test token validation with invalid signature
- Test token validation with malformed token

**`internal/auth/password_test.go`:**
- Test password hashing
- Test correct password verification
- Test incorrect password verification
- Test empty password handling

**`internal/auth/middleware_test.go`:**
- Test valid token allows access
- Test expired token returns 401
- Test missing token returns 401
- Test user without site access returns 403
- Test global admin can access any site

### Integration Tests

**Login Flow:**
- POST `/admin/login` with valid credentials → 302 redirect + cookie set
- POST `/admin/login` with invalid credentials → 401
- POST `/admin/login` with valid user but wrong site → 403

**Protected Routes:**
- GET `/admin/dashboard` without cookie → 401 redirect to login
- GET `/admin/dashboard` with valid cookie → 200
- GET `/admin/dashboard` with expired cookie → 401

**Logout:**
- POST `/admin/logout` → cookie cleared, redirect to login

### Manual Testing Checklist

- [ ] Login with valid credentials works
- [ ] Login with wrong password fails
- [ ] Access admin page without login redirects to login
- [ ] Logout clears session
- [ ] Cookie expires after 8 hours
- [ ] Global admin can access any site
- [ ] Regular user cannot access other sites
- [ ] Rate limiting prevents brute force

## Future Enhancements

1. **Visible login link** - Add "Admin" or "Login" link to site navigation menu (prevent `/wp-admin` confusion)
2. **"Remember me" option** - Optional 30-day token with refresh mechanism
3. **Two-factor authentication** - TOTP for high-security sites
4. **Password reset flow** - Email-based password recovery
5. **Audit log** - Track who logged in when
6. **Session management UI** - View/revoke active sessions
7. **API keys** - For future headless/API access

## Success Criteria

- Users can log in with email/password
- JWT tokens are properly validated
- Unauthorized users cannot access admin panel
- Global admins can access all sites
- Regular users can only access their sites
- Sessions expire after 8 hours
- Rate limiting prevents brute force
- All tests pass
