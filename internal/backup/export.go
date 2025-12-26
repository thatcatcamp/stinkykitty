package backup

// SiteExporter handles site-specific exports
type SiteExporter struct {
	BackupPath string
}

// NewSiteExporter creates a new site exporter
func NewSiteExporter(backupPath string) *SiteExporter {
	return &SiteExporter{
		BackupPath: backupPath,
	}
}
