package common

import "errors"

var (
	// common errors
	ErrSuccess             = New(0, "ok")
	ErrUnknown             = New(1, "unknown error")
	ErrBadRequest          = New(400, "Bad Request")
	ErrUnauthorized        = New(401, "Unauthorized")
	ErrNotFound            = New(404, "Not Found")
	ErrInternalServerError = New(500, "Internal Server Error")

	// creator errors
	ErrEmailAlreadyUse       = New(10001, "the email is already in use.")
	ErrNameAlreadyUse        = New(10002, "the name is already in use.")
	ErrEmailOrNameAlreadyUse = New(10003, "the email or name is already in use.")
	ErrInvalidCredential     = New(10004, "incorrect password.")
	// Account or password is incorrect
	ErrInvalidAccountOrPassword = New(10004, "invalid account or password.")

	ErrCreatorNotFound = New(10005, "creator not found.")

	// site errors
	ErrSiteAlreadyExist = New(20001, "site already exist.")
	// Site does not exist
	ErrSiteNotFound = New(20002, "site not found.")
	// Invalid guide name
	ErrInvalidGuides = New(20003, "invalid guides.")
	// A user can have at most one site
	ErrTooManySites = New(20004, "too many sites.")
	// Path is not valid
	ErrInvalidPath = New(20005, "invalid path.")
	// Invalid custom domain
	ErrInvalidCustomDomain = New(20006, "invalid custom domain.")
	// No data to publish
	ErrNoDataToPublish = New(20007, "no data to publish.")
	// Sensitive word detected
	ErrSensitiveWordDetected = New(20008, "sensitive word detected")
	// site must have template to initialize pages
	ErrSiteTemplateRequired = New(20009, "site template required")
	// Site page config not found
	ErrSitePageConfigNotFound = New(20010, "site page config not found")

	//playlist errors
	ErrPlaylistNotFound = New(30001, "playlist not found.")
	// Maximum 10 playlists can be queried for details at once
	ErrTooManyPlaylistsGetDetail = New(30002, "too many playlists get detail.")
	// Video list record has been updated, please re-fetch
	ErrPlaylistUpdated = New(30003, "playlist updated, please update it again.")

	//video errors
	ErrVideoNotFound = New(40001, "video not found.")
	ErrNoFilesFound  = New(40002, "no video files found.")
	// Number of files uploaded exceeds limit
	ErrTooManyFiles = New(40003, "too many files.")
	// Number of videos added to playlist exceeds limit
	ErrTooManyVideosAddToPalaylist = New(40004, "too many videos added to the playlist.")
	// Video already exists in playlist
	ErrVideoAlreadyInPlaylist = New(40005, "video already in the playlist.")
	// Maximum 10 videos can be uploaded at once
	ErrTooManyVideosUpload = New(40006, "too many videos uploaded.")
	// Maximum 10 videos can be queried for details at once
	ErrTooManyVideosGetDetail = New(40007, "too many videos get detail.")

	// user errors
	ErrUserNotFound        = New(5000, "user not found")
	ErrUserProfileNotFound = New(5001, "user profile not found")
	// Email already registered
	ErrEmailAlreadyRegistered = New(5002, "email already registered")
	// User not activated
	ErrUserNotActive = New(5003, "user not active")
	// User banned
	ErrUserBanned       = New(5004, "user banned")
	ErrUserAuthNotFound = New(5005, "user auth not found")

	// Ads and payment
	// Ad with the same name already exists
	ErrAdNameAlreadyExist           = New(6001, "ad name already exist")
	ErrTestConfigNotAvailable       = New(6002, "test configuration file not available")
	ErrPaymentProviderNotConfigured = New(6003, "payment provider not configured")
	ErrCreateStripeProduct          = New(6004, "failed to create stripe product")
	ErrCreateStripePrice            = New(6005, "failed to create stripe price")
	ErrCreateStripePaymentIntent    = New(6006, "failed to create stripe payment intent")
	ErrCreateStripeCheckoutSession  = New(6007, "failed to create stripe checkout session")
	ErrResourceNotActive            = New(6008, "requested resource is not active")
	ErrInsufficientCoins            = New(6009, "insufficient coins balance")

	// translate error

	ErrHandleTranslateFailed   = New(7001, "handle translate failed")
	ErrValidateTranslateResult = New(7002, "validate translate result failed")

	ErrHookRunFailed        = New(8001, "hook run failed")
	ErrHookActiveCallFailed = New(8001, "hook active call failed")
)

type Error struct {
	Code    int
	Message string
}

var errorCodeMap = map[error]int{}

func New(code int, msg string) error {
	err := errors.New(msg)
	errorCodeMap[err] = code
	return err
}

func (e Error) Error() string {
	return e.Message
}

func GetErrorCodeMap() map[error]int {
	return errorCodeMap
}
