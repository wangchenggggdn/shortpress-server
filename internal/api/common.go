package api

// IDListResponseData Playlist video list data
type IDListResponseData[T any] struct {
	Total    int  `json:"total" example:"100"`    // Total number of videos
	Page     int  `json:"page" example:"1"`       // Current page number
	PageSize int  `json:"pageSize" example:"20"`  // Items per page
	HasMore  bool `json:"hasMore" example:"true"` // Whether there are more items
	Items    []T  `json:"items"`                  // ID information list
}

// type IDListResponse struct {
// 	Response
// 	Data *IDListResponseData `json:"data"`
// }
