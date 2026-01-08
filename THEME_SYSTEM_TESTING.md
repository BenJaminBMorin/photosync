# Theme System Testing Checklist

## Code Review Status: ✅ COMPLETE

### Fixed Issues
1. **CSS Variable Naming** - Fixed mismatch between frontend and backend
   - `--font-base` → `--font-family-base`
   - `--radius-sm` → `--border-radius-sm`
   - `--border-default` → `--border-color`
   - Added missing variables: `--bg-tertiary-hover`, `--color-error-dark`, `--shadow-color`, `--hover-scale`

### Database Migration Readiness: ✅ READY

**PostgreSQL** (`/server-go/internal/repository/postgres.go`)
- ✅ `themes` table with JSONB properties (lines 277-286)
- ✅ `user_preferences` table (lines 292-297)
- ✅ Indexes on themes table (lines 288-289)
- ✅ `theme_source` column migration for collections (line 353)

**SQLite** (`/server-go/internal/repository/sqlite.go`)
- ✅ `themes` table with TEXT (JSON) properties (lines 248-257)
- ✅ `user_preferences` table (lines 259-265)
- ✅ `theme_source` column migration with pragma check (lines 280-295)

**Seed Data** (`/server-go/internal/repository/seed_themes.go`)
- ✅ 5 system themes: Dark, Light, Minimal, Gallery, Magazine
- ✅ Called on server startup (main.go lines 212-216)

### Backend Architecture: ✅ VERIFIED

**Services**
- ✅ `ThemeService` with caching (`theme_service.go`)
- ✅ `ThemeCache` with 1-hour TTL (`theme_cache.go`)
- ✅ `CollectionService` updated with theme resolution
- ✅ CSS generation from ~100 theme properties

**Handlers**
- ✅ `ThemeHandler` for theme CRUD operations
- ✅ `UserHandler` for user preferences
- ✅ `CollectionHandler` updated for theme inheritance

**Routing** (main.go)
- ✅ Public theme routes (lines 291-293)
- ✅ User preferences routes (lines 364-365)
- ✅ Admin theme management routes (lines 428-433)

### Frontend Implementation: ✅ COMPLETE

**Admin Theme Builder** (`/web/admin/theme-builder.html`)
- ✅ Multi-section editor (Colors, Typography, Layout, Effects, Preview)
- ✅ Safe DOM methods (no XSS vulnerabilities)
- ✅ CRUD operations via API

**Gallery Dynamic Theming** (`/web/index.html`)
- ✅ Loads theme CSS from `/api/themes/:id/css`
- ✅ Fetches user preference from `/api/users/me/preferences`
- ✅ Dynamic CSS injection

**Collections Theme Inheritance** (`/web/collections.html`)
- ✅ Theme source selector (inherit vs explicit)
- ✅ Dynamic theme loading
- ✅ Save/edit with `themeSource` field
- ✅ CSS variables replaced throughout

## Testing Plan

### Phase 1: Compilation & Startup
```bash
cd server-go
go build -o photosync-server ./cmd/server
./photosync-server
```

**Expected Results:**
- ✅ No compilation errors
- ✅ Database migrations run successfully
- ✅ "System themes initialized" message in logs
- ✅ Server starts on configured port

### Phase 2: Database Verification
```sql
-- PostgreSQL
SELECT id, name, is_system FROM themes;
SELECT * FROM user_preferences LIMIT 1;
SELECT theme, theme_source FROM collections LIMIT 5;

-- SQLite
sqlite3 photosync.db
SELECT id, name, is_system FROM themes;
SELECT * FROM user_preferences LIMIT 1;
SELECT theme, theme_source FROM collections LIMIT 5;
```

**Expected Results:**
- ✅ 5 system themes present (dark, light, minimal, gallery, magazine)
- ✅ Tables created with correct schema
- ✅ Existing collections have `theme_source='explicit'`

### Phase 3: API Endpoint Testing

**Public Theme Endpoints (no auth):**
```bash
# List all themes
curl http://localhost:5050/api/themes

# Get specific theme
curl http://localhost:5050/api/themes/dark

# Get theme CSS (should return valid CSS)
curl http://localhost:5050/api/themes/dark/css
```

**User Preferences (requires session auth):**
```bash
# Get preferences
curl -b cookies.txt http://localhost:5050/api/users/me/preferences

# Update preferences
curl -b cookies.txt -X PUT \
  -H "Content-Type: application/json" \
  -d '{"globalThemeId":"dark"}' \
  http://localhost:5050/api/users/me/preferences
```

**Admin Theme Management (requires admin auth):**
```bash
# List all themes (admin view)
curl -b admin-cookies.txt http://localhost:5050/api/admin/themes

# Create custom theme
curl -b admin-cookies.txt -X POST \
  -H "Content-Type: application/json" \
  -d @custom-theme.json \
  http://localhost:5050/api/admin/themes

# Update theme
curl -b admin-cookies.txt -X PUT \
  -H "Content-Type: application/json" \
  -d @updated-theme.json \
  http://localhost:5050/api/admin/themes/custom-ocean

# Delete theme
curl -b admin-cookies.txt -X DELETE \
  http://localhost:5050/api/admin/themes/custom-ocean
```

### Phase 4: Frontend Integration Testing

**1. Main Gallery Theme Loading**
- Navigate to `http://localhost:5050`
- Login via push notification
- Open browser DevTools → Network tab
- Verify `/api/users/me/preferences` is called
- Verify `/api/themes/{id}/css` is called
- Check Elements tab for `<style id="dynamic-theme">`
- Verify CSS variables are injected in `:root`

**2. Theme Builder (Admin)**
- Navigate to `http://localhost:5050/admin`
- Click "Theme Builder" in settings
- Verify all system themes load in dropdown
- Test creating a new theme:
  - Change colors in Colors section
  - Change fonts in Typography section
  - Check live preview updates
  - Click "Save Theme"
- Verify new theme appears in dropdown
- Test duplicating a theme
- Test deleting a custom theme (should fail for system themes)

**3. Collections Theme Inheritance**
- Navigate to `http://localhost:5050/collections`
- Click "New Collection"
- Verify theme source dropdown shows:
  - "Use my default theme" (inherit)
  - "Custom theme for this collection" (explicit)
- Select "Custom theme" and choose a theme
- Create collection and verify it uses the selected theme
- Edit collection, change to "Use my default theme"
- Save and verify it now inherits global theme

**4. Theme Switching**
- In admin panel, change your global theme preference
- Navigate back to main gallery
- Verify new theme is loaded
- Create a new collection with "inherit"
- View the collection and verify it uses the global theme
- Change global theme again
- Verify inherited collection updates, but explicit collections don't

### Phase 5: Cache Testing

**1. CSS Generation Performance**
```bash
# First request (cache miss)
time curl http://localhost:5050/api/themes/dark/css

# Second request (cache hit - should be faster)
time curl http://localhost:5050/api/themes/dark/css

# Check logs for cache hit/miss messages
```

**2. Cache Invalidation**
- Update a theme via admin API
- Request theme CSS
- Verify CSS reflects the update (cache was cleared)

### Phase 6: Error Handling

**1. Invalid Theme ID**
```bash
curl http://localhost:5050/api/themes/nonexistent/css
# Expected: 404 or fallback to default theme
```

**2. Invalid Theme Properties**
```bash
# Try to create theme with missing required fields
curl -b admin-cookies.txt -X POST \
  -H "Content-Type: application/json" \
  -d '{"name":"Bad Theme"}' \
  http://localhost:5050/api/admin/themes
# Expected: 400 Bad Request with validation error
```

**3. Delete System Theme**
```bash
curl -b admin-cookies.txt -X DELETE \
  http://localhost:5050/api/admin/themes/dark
# Expected: 403 Forbidden or similar error
```

**4. Collection with Missing Theme**
- Set collection theme to non-existent ID in database
- Load the collection
- Verify fallback to default theme

## Known Limitations

1. **Go Compiler Not Available**: Cannot compile and test locally yet
2. **Hardcoded Fallback Values**: Some CSS variables have hardcoded fallbacks in frontend for offline mode
3. **Theme Preview Colors**: Theme builder uses simplified color map for previews

## Success Criteria

- ✅ All code files created and reviewed
- ⏳ Server compiles without errors
- ⏳ Database migrations run successfully
- ⏳ All API endpoints return expected responses
- ⏳ Frontend loads themes dynamically
- ⏳ Theme builder creates and manages themes
- ⏳ Collection inheritance works correctly
- ⏳ CSS cache improves performance
- ⏳ No XSS vulnerabilities
- ⏳ Zero breaking changes for existing users

## Next Steps

1. **Compile and test** on a system with Go installed
2. **Start the server** and verify startup logs
3. **Run database verification** queries
4. **Test each API endpoint** with curl
5. **Manually test frontend** in browser
6. **Performance testing** with ab or similar tool
7. **Create user documentation** for theme builder
8. **Update README** with theme system documentation

## Files Modified/Created

### Backend (Go)
- ✅ `/server-go/internal/models/theme.go` (NEW)
- ✅ `/server-go/internal/models/user_preferences.go` (NEW)
- ✅ `/server-go/internal/models/collection.go` (UPDATED)
- ✅ `/server-go/internal/repository/theme_repository.go` (NEW)
- ✅ `/server-go/internal/repository/user_preferences_repository.go` (NEW)
- ✅ `/server-go/internal/repository/seed_themes.go` (NEW)
- ✅ `/server-go/internal/repository/postgres.go` (UPDATED)
- ✅ `/server-go/internal/repository/sqlite.go` (UPDATED)
- ✅ `/server-go/internal/services/theme_service.go` (NEW)
- ✅ `/server-go/internal/services/theme_cache.go` (NEW)
- ✅ `/server-go/internal/services/collection_service.go` (UPDATED)
- ✅ `/server-go/internal/handlers/theme_handler.go` (NEW)
- ✅ `/server-go/internal/handlers/user_handler.go` (NEW)
- ✅ `/server-go/cmd/server/main.go` (UPDATED)

### Frontend (HTML/JS/CSS)
- ✅ `/server-go/web/admin/theme-builder.html` (NEW)
- ✅ `/server-go/web/admin/index.html` (UPDATED)
- ✅ `/server-go/web/index.html` (UPDATED)
- ✅ `/server-go/web/collections.html` (UPDATED)

### Documentation
- ✅ Plan file at `/Users/benjamin/.claude/plans/iridescent-wiggling-narwhal.md`
- ✅ This testing checklist

---

**Status**: Ready for testing when Go compiler is available
**Last Updated**: 2026-01-07
