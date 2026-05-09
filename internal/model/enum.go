package model

// Video status enumeration
const (
	// Initial state with no business meaning
	VideoStatusInvalid int8 = 0
	// Unpublished
	VideoStatusUnpublished int8 = 1
	// Published
	VideoStatusPublished int8 = 2
	// Offline
	VideoStatusOffline int8 = 3
	// Deleted
	VideoStatusDeleted int8 = 127
)

// Video upload status enumeration
const (
	// Initial state with no business meaning
	VideoUploadStatusInvalid int8 = 0
	// Not uploaded
	VideoUploadStatusUnuploaded int8 = 1
	// Uploading
	VideoUploadStatusUploading int8 = 2
	// Upload failed
	VideoUploadStatusUploadFailed int8 = 3
	// Upload canceled
	VideoUploadStatusUploadCanceled int8 = 4
	// Upload successful
	VideoUploadStatusUploadSuccess int8 = 5
)

// Playlist status enumeration
const (
	// Initial state with no business meaning
	PlaylistStatusInvalid int8 = 0
	// Unpublished
	PlaylistStatusUnpublished int8 = 1
	// Published
	PlaylistStatusPublished int8 = 2
	// Offline
	PlaylistStatusOffline int8 = 3
	// Deleted
	PlaylistStatusDeleted int8 = 127
)

// Site status enumeration
const (
	// Initial state with no business meaning
	SiteStatusInvalid int8 = 0
	// Unpublished
	SiteStatusUnpublished int8 = 1
	// Published
	SiteStatusPublished int8 = 2
	// Offline
	SiteStatusOffline int8 = 3
	// Deleted
	SiteStatusDeleted int8 = 127
)

// Define video list sorting method constants (0:Create time descending 1:Name sorting)
const (
	VideoSortByCreatedAtDesc = 0 // Sort by creation time descending
	VideoSortTitleAsc        = 1 // Sort by title ascending
	VideoSortByCreatedAtAsc  = 2 // Sort by creation time ascending
	VideoSortByTitleDesc     = 3 // Sort by title descending
)

const (
	// Video source upload type constants
	VideoSourceProvider = "local" // Local upload provider
)

const (
	VideoSourceStatusEnabled  int8 = 1 // Enabled
	VideoSourceStatusDisabled int8 = 2 // Disabled
)
const (
	// Video source type constants
	VideoSourceTypeLocalFile int8 = 1 // Local file
	VideoSourceTypeHTTPLink  int8 = 2 // HTTP link
	VideoSourceTypeEmbed     int8 = 3 // Embed code
)
