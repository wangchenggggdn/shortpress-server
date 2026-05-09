package api

// AdUnit represents the data structure for an ad unit
type AdUnit struct {
	AdID      string `json:"id"`
	SiteID    string `json:"siteId" binding:"required"`
	Name      string `json:"name" binding:"required"`
	AdNetwork string `json:"adNetwork" binding:"required"`
	Page      string `json:"page" binding:"required"`
	Format    int    `json:"format"`
	Status    int    `json:"status"`
	Frequency int    `json:"frequency"`
	ClientID  string `json:"clientId"`
	UnitID    string `json:"unitId"`
}

// AdUnitListResponseData defines ad unit list response data
type AdUnitListResponseData struct {
	Items []*AdUnit `json:"items"` // Ad unit list
}

// AdUnitConf represents the ad unit configuration for client-side
type AdUnitConf struct {
	Page      string                 `json:"page"`      // Ad location on the page
	AdNetwork string                 `json:"adNetwork"` // Ad network platform (e.g., AdMob, Facebook)
	Format    int                    `json:"format"`    // Ad format type
	Status    int                    `json:"status"`
	Conf      map[string]interface{} `json:"conf"`    // Ad configuration values
	ShowStg   map[string]interface{} `json:"showStg"` // Show strategy values
}
