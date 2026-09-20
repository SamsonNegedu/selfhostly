# Compose Configuration Versioning & Rollback System

## Overview

This document describes the comprehensive versioning and rollback system for Docker Compose configurations implemented in the selfhostly platform.

## Features

### Automatic Versioning
- **Initial Version**: When an app is created, version 1 is automatically saved
- **Change Tracking**: Every compose file update creates a new version
- **Version Metadata**: Each version tracks:
  - Version number (sequential integer)
  - Compose file content
  - Change reason (optional description)
  - Changed by (username of authenticated user)
  - Creation timestamp
  - Rollback information (if version was created from a rollback)
  - Current status (whether this is the active version)

### Version History
- **Timeline View**: Visual timeline of all versions in reverse chronological order
- **Version Details**: Each version displays:
  - Version number with visual indicator
  - Current version badge
  - Rollback indicator (if applicable)
  - Change reason/description
  - Timestamp with relative time (e.g., "2h ago")
  - Changed by username
- **Actions**: 
  - View version content
  - Rollback to any previous version

### Rollback Capability
- **Safe Rollback**: Creates a new version with previous content (non-destructive)
- **Rollback Metadata**: Tracks which version was rolled back from
- **Container Update**: After rollback, prompts to update containers
- **History Preserved**: All versions remain available for future reference

## Architecture

### Database Schema

```sql
CREATE TABLE compose_versions (
    id TEXT PRIMARY KEY,
    app_id TEXT NOT NULL,
    version INTEGER NOT NULL,
    compose_content TEXT NOT NULL,
    change_reason TEXT,
    changed_by TEXT,
    is_current INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    rolled_back_from INTEGER,
    UNIQUE(app_id, version),
    FOREIGN KEY (app_id) REFERENCES apps(id) ON DELETE CASCADE
);

CREATE INDEX idx_compose_versions_app_id ON compose_versions(app_id);
CREATE INDEX idx_compose_versions_is_current ON compose_versions(app_id, is_current);
```

### Backend Implementation

#### Models (`internal/db/models.go`)

```go
type ComposeVersion struct {
    ID             string     `json:"id" db:"id"`
    AppID          string     `json:"app_id" db:"app_id"`
    Version        int        `json:"version" db:"version"`
    ComposeContent string     `json:"compose_content" db:"compose_content"`
    ChangeReason   *string    `json:"change_reason" db:"change_reason"`
    ChangedBy      *string    `json:"changed_by" db:"changed_by"`
    IsCurrent      bool       `json:"is_current" db:"is_current"`
    CreatedAt      time.Time  `json:"created_at" db:"created_at"`
    RolledBackFrom *int       `json:"rolled_back_from" db:"rolled_back_from"`
}
```

#### Database Methods (`internal/db/db.go`)

- `CreateComposeVersion()` - Create a new version
- `GetComposeVersionsByAppID()` - Get all versions for an app
- `GetComposeVersion()` - Get a specific version
- `GetCurrentComposeVersion()` - Get the active version
- `GetLatestVersionNumber()` - Get the latest version number
- `MarkVersionAsCurrent()` - Set a version as current
- `MarkAllVersionsAsNotCurrent()` - Clear current flag from all versions

#### API Endpoints (`internal/http/routes.go`)

- `GET /api/apps/:id/compose/versions` - List all versions
- `GET /api/apps/:id/compose/versions/:version` - Get specific version
- `POST /api/apps/:id/compose/rollback/:version` - Rollback to version

#### Auto-Versioning Logic (`internal/http/app.go`)

**On App Creation:**
```go
// Create initial compose version (version 1)
initialReason := "Initial version"
initialVersion := db.NewComposeVersion(app.ID, 1, app.ComposeContent, &initialReason, changedBy)
db.CreateComposeVersion(initialVersion)
```

**On App Update:**
```go
// Check if compose content has changed
composeChanged := composeContent != app.ComposeContent

if composeChanged {
    latestVersion, _ := db.GetLatestVersionNumber(id)
    db.MarkAllVersionsAsNotCurrent(id)
    
    updateReason := "Compose file updated"
    newVersion := db.NewComposeVersion(id, latestVersion+1, app.ComposeContent, &updateReason, changedBy)
    db.CreateComposeVersion(newVersion)
}
```

**On Rollback:**
```go
// Create new version with rolled-back content
newVersionNumber := currentVersionNumber + 1
changeReason := fmt.Sprintf("Rolled back to version %d", targetVersion)
newVersion := db.NewComposeVersion(id, newVersionNumber, targetComposeVersion.ComposeContent, &changeReason, changedBy)
newVersion.RolledBackFrom = &targetVersion

db.MarkAllVersionsAsNotCurrent(id)
db.CreateComposeVersion(newVersion)
db.UpdateApp(app) // Update app with rolled-back content
dockerManager.WriteComposeFile(app.Name, app.ComposeContent) // Update file on disk
```

### Frontend Implementation

#### Types (`web/src/shared/types/api.ts`)

```typescript
export interface ComposeVersion {
  id: string;
  app_id: string;
  version: number;
  compose_content: string;
  change_reason?: string | null;
  changed_by?: string | null;
  is_current: boolean;
  created_at: string;
  rolled_back_from?: number | null;
}
```

#### API Hooks (`web/src/shared/services/api.ts`)

- `useComposeVersions(appId)` - Fetch all versions
- `useComposeVersion(appId, version)` - Fetch specific version
- `useRollbackToVersion(appId)` - Rollback mutation

#### Components

**ComposeVersionHistory Component** (`web/src/features/app-details/components/ComposeVersionHistory.tsx`)
- Displays version timeline
- Shows version metadata
- Handles rollback confirmation
- Responsive design (mobile + desktop)

**ComposeEditor Component** (`web/src/features/app-details/components/ComposeEditor.tsx`)
- Integrated version history sidebar
- Auto-updates on rollback
- Mobile-friendly toggle for version history
- Side-by-side layout on desktop

## User Workflow

### Creating an App
1. User creates app with initial compose file
2. System automatically creates **version 1**
3. Version is marked as current

### Editing Compose File
1. User edits compose content in editor
2. User clicks "Save Changes"
3. System detects changes and creates **new version**
4. New version marked as current
5. Previous version remains in history

### Rolling Back
1. User views version history
2. User clicks "Rollback" on desired version
3. System shows confirmation dialog
4. On confirm:
   - Creates **new version** with old content
   - Marks rollback metadata
   - Updates app compose content
   - Updates compose file on disk
5. User prompted to update containers

### Viewing History
- **Desktop**: Version history shown in sidebar
- **Mobile**: Toggleable version history panel
- Timeline shows:
  - All versions newest to oldest
  - Visual indicators for current version
  - Rollback badges
  - Timestamps and authors

## Benefits

### For Users
- **Safety**: Never lose a working configuration
- **Confidence**: Experiment knowing you can rollback
- **Traceability**: See who changed what and when
- **Auditability**: Complete history of all changes

### For System
- **Non-destructive**: All operations preserve history
- **Cascading Deletes**: Versions deleted when app deleted
- **Performance**: Indexed queries for fast retrieval
- **Storage**: Minimal overhead (text compression possible)

## Best Practices

### Version Management
- Keep descriptive change reasons
- Review version history before making major changes
- Test changes before rolling back in production
- Remember to update containers after rollback

### Database Maintenance
- Versions automatically cleaned up when app deleted
- Consider periodic archival for very old versions
- Monitor database size if many versions accumulate

## Future Enhancements

Potential improvements to consider:

1. **Diff Viewer**: Show differences between versions
2. **Version Tags**: Mark important versions (e.g., "stable", "production")
3. **Version Notes**: Add detailed notes to versions
4. **Scheduled Rollbacks**: Schedule rollback for specific time
5. **Version Comparison**: Compare any two versions side-by-side
6. **Auto-save**: Create versions on timer or after N edits
7. **Version Limits**: Configure max versions per app
8. **Compression**: Compress old version content
9. **Export/Import**: Export version history with app
10. **Merge Conflicts**: Handle concurrent edits

## API Reference

### List Versions
```
GET /api/apps/{appId}/compose/versions
```
Returns array of all versions for the app, ordered by version DESC.

### Get Specific Version
```
GET /api/apps/{appId}/compose/versions/{version}
```
Returns the specified version details.

### Rollback to Version
```
POST /api/apps/{appId}/compose/rollback/{version}
Content-Type: application/json

{
  "change_reason": "Optional reason for rollback"
}
```
Creates new version with content from target version.

## Security Considerations

- User authentication required for all operations
- User name captured for audit trail
- Versions tied to app via foreign key
- Cascading delete ensures no orphaned versions
- SQL injection prevented via parameterized queries

## Implementation Notes

- Version numbers start at 1 and increment
- Rollback creates **new version**, doesn't modify old ones
- Current version tracked via `is_current` flag
- Only one version can be current at a time
- Database transaction ensures atomic current version updates
- Compose file on disk always matches current version
- Frontend auto-refreshes after rollback

## Testing Checklist

- [ ] Create app creates version 1
- [ ] Update compose creates new version
- [ ] No version created if content unchanged
- [ ] Rollback creates new version with old content
- [ ] Rollback metadata correctly set
- [ ] Current version flag properly managed
- [ ] Version history displays correctly
- [ ] Mobile version history toggle works
- [ ] User names captured when authenticated
- [ ] Timestamps accurate
- [ ] Cascade delete removes versions
- [ ] Concurrent edits handled safely

## Troubleshooting

### Version not created
- Check if compose content actually changed
- Verify database connection
- Check logs for errors

### Rollback fails
- Ensure target version exists
- Check app permissions
- Verify disk space for file write

### History not showing
- Check API endpoint accessibility
- Verify authentication
- Check browser console for errors

## Conclusion

This versioning system provides a robust, production-ready solution for managing Docker Compose configurations with full history tracking and safe rollback capabilities. The implementation follows best practices for data integrity, user experience, and system maintainability.
