package server

import (
	"net/http"
	"path/filepath"

	"shortpress-server/docs"
	"shortpress-server/internal/api"
	"shortpress-server/internal/handler"
	"shortpress-server/internal/middleware"
	"shortpress-server/pkg/jwt"
	"shortpress-server/pkg/log"
	apphttp "shortpress-server/pkg/server/http"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/viper"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func NewHTTPServer(
	logger *log.Logger,
	conf *viper.Viper,
	jwt *jwt.JWT,
	creatorHandler *handler.CreatorHandler,
	siteHandler *handler.SiteHandler,
	sitePageConfigHandler *handler.SitePageConfigHandler,
	videoHandler *handler.VideoHandler,
	playlistHandler *handler.PlaylistHandler,
	clientPlayerHandler *handler.ClientPlayerHandler,
	adsHandler *handler.AdsHandler,
	userHandler *handler.UserHandler,
	paymentHandler *handler.PaymentHandler,
	coinsHandler *handler.CoinsHandler,
	coinsInternalHandler *handler.CoinsInternalHandler,
	subscriptionHandler *handler.SubscriptionHandler,
	analyticsHandler *handler.AnalyticsHandler,
	pageBuilderHandler *handler.PagesBuilderHandler,
	a2eHandler *handler.A2EHandler,
	promptHandler *handler.PromptHandler,
	pluginHandler *handler.PluginHandler,
	iapHandler *handler.IapHandler,
) *apphttp.Server {
	gin.SetMode(gin.DebugMode)

	s := apphttp.NewServer(
		gin.Default(),
		logger,
		apphttp.WithServerHost(conf.GetString("http.host")),
		apphttp.WithServerPort(conf.GetInt("http.port")),
	)

	// 本地/单机：image_host 指向本服务端口时，需把磁盘上的 videolib、res 映射为 URL 路径（生产环境常由 Nginx /media 反代，此处补齐 go run 场景）
	if root := conf.GetString("storage.local.path"); root != "" {
		root = filepath.Clean(root)
		s.StaticFS("/videolib", http.Dir(filepath.Join(root, "videolib")))
		s.StaticFS("/res", http.Dir(filepath.Join(root, "res")))
	}

	// swagger doc
	docs.SwaggerInfo.BasePath = "/"
	s.GET("/swagger/*any", ginSwagger.WrapHandler(
		swaggerfiles.Handler,
		ginSwagger.DefaultModelsExpandDepth(-1),
		ginSwagger.PersistAuthorization(true),
	))

	s.Use(
		middleware.CORSMiddleware(),
		middleware.ResponseLogMiddleware(logger),
		middleware.RequestLogMiddleware(logger),
		middleware.PrometheusMiddleware(logger),
	)

	s.GET("/", func(ctx *gin.Context) {
		logger.WithContext(ctx).Info("hello")
		api.HandleSuccess(ctx, map[string]interface{}{
			":)": "Hello shortpress user",
		})
	})
	s.GET("/metrics", gin.WrapH(promhttp.Handler()))

	s.MaxMultipartMemory = 1024 << 20
	apiGroup := s.Group("/api")
	{
		registerCreatorRoutes(apiGroup, creatorHandler, jwt, logger)
		registerSiteRoutes(apiGroup, siteHandler, jwt, logger)
		registerSitePageConfigRoutes(apiGroup, sitePageConfigHandler, jwt, logger)
		registerPageBuilderRoutes(apiGroup, pageBuilderHandler, jwt, logger)
		registerVideoRoutes(apiGroup, videoHandler, jwt, logger)
		registerPlaylistRoutes(apiGroup, playlistHandler, jwt, logger)
		registerAdsRoutes(apiGroup, adsHandler, jwt, logger)
		registerPaymentRoutes(apiGroup, paymentHandler, coinsHandler, subscriptionHandler, jwt, logger)
		registerAnalyticsRoutes(apiGroup, analyticsHandler, jwt, logger)
		registerUserRoutes(apiGroup, userHandler, jwt, logger)
		registerA2ERoutes(apiGroup, a2eHandler, jwt, logger)
		registerPromptRoutes(apiGroup, promptHandler, jwt, logger)
		registerClientRoutes(apiGroup, clientPlayerHandler, paymentHandler, coinsHandler, subscriptionHandler, jwt, logger)
		registerPluginsRoutes(apiGroup, pluginHandler, jwt, logger, conf)
		registerInternalCoinsRoutes(apiGroup, coinsInternalHandler, jwt, logger)
		registerInternalVideoRoutes(apiGroup, videoHandler)
		registerIapRoutes(apiGroup, iapHandler, jwt, logger)
	}
	return s
}

func registerCreatorRoutes(apiGroup *gin.RouterGroup, creatorHandler *handler.CreatorHandler, jwt *jwt.JWT, logger *log.Logger) {
	creatorGroup := apiGroup.Group("/creator")
	{
		creatorGroup.POST("/register", creatorHandler.CreatorRegister)
		creatorGroup.POST("/resetpwd", creatorHandler.CreatorRestPwd)
		creatorGroup.POST("/login", creatorHandler.CreatorLogin)

		// auth
		creatorAuthRouter := creatorGroup.Group("").Use(middleware.StrictAuth(jwt, logger))
		{
			creatorAuthRouter.GET("/profile", creatorHandler.CreatorProfile)
			creatorAuthRouter.GET("/stats", creatorHandler.Stats)
			creatorAuthRouter.POST("/upload-file", creatorHandler.UploadFile)
			creatorAuthRouter.POST("/complete-guides", creatorHandler.CompleteGuides)
		}
	}
}

func registerSiteRoutes(apiGroup *gin.RouterGroup, siteHandler *handler.SiteHandler, jwt *jwt.JWT, logger *log.Logger) {
	siteGroup := apiGroup.Group("/site").Use(middleware.StrictAuth(jwt, logger))
	{
		siteGroup.POST("/create", siteHandler.Create)
		siteGroup.GET("/path/valid", siteHandler.PathValid)

		siteGroup.GET("/get", siteHandler.Get)
		siteGroup.GET("/batch-get", siteHandler.BatchGet)
		siteGroup.POST("/modify", siteHandler.Modify)
		siteGroup.POST("/add-playlists", siteHandler.AddPlaylists)
		siteGroup.POST("/remove-playlists", siteHandler.RemovePlaylists)
		siteGroup.GET("/list", siteHandler.List)
		siteGroup.GET("/user/list", siteHandler.UserList)
		siteGroup.POST("/user/change/status", siteHandler.UserChangeStatus)
		siteGroup.POST("/user/reset/password", siteHandler.UserResetPassword)
		siteGroup.GET("/user/info", siteHandler.UserInfo)
	}
}

func registerVideoRoutes(apiGroup *gin.RouterGroup, videoHandler *handler.VideoHandler, jwt *jwt.JWT, logger *log.Logger) {
	videoGroup := apiGroup.Group("/video").Use(middleware.StrictAuth(jwt, logger))
	{
		videoGroup.POST("/upload", videoHandler.Upload)
		videoGroup.POST("/upload-subtitle", videoHandler.UploadSubtitle)
		videoGroup.POST("/sources/add", videoHandler.AddSources)

		videoGroup.POST("/replace", videoHandler.Replace)
		videoGroup.POST("/modify", videoHandler.Modify)
		videoGroup.POST("/delete", videoHandler.Delete)
		videoGroup.GET("/search", videoHandler.Search)
		videoGroup.GET("/list", videoHandler.List)

		//仅用于生重新生成封面图 ，临时使用，后续需要删除或者禁用这个结构。 目标是当前用户所属于的某get website的下的视频重新生成。支持多playlistid
		//videoGroup.GET("/regenerate/cover", videoHandler.RegenerateCover)

	}
	videoGroupNoAuth := apiGroup.Group("/video").Use(middleware.RequireSiteID(logger))
	{
		videoGroupNoAuth.GET("/batch-get", videoHandler.BatchGet)
		videoGroupNoAuth.GET("/get", videoHandler.Get)
	}

}

func registerPlaylistRoutes(apiGroup *gin.RouterGroup, playlistHandler *handler.PlaylistHandler, jwt *jwt.JWT, logger *log.Logger) {
	playlistGroup := apiGroup.Group("/playlist").Use(middleware.StrictAuth(jwt, logger))
	{
		playlistGroup.POST("/create", playlistHandler.Create)
		playlistGroup.GET("/get", playlistHandler.Get)
		playlistGroup.POST("/modify", playlistHandler.Modify)
		playlistGroup.POST("/delete", playlistHandler.Delete)
		playlistGroup.GET("/videos", playlistHandler.VideosV2)
		playlistGroup.POST("/add-videos", playlistHandler.AddVideos)
		playlistGroup.POST("/remove-videos", playlistHandler.RemoveVideos)
		playlistGroup.GET("/list", playlistHandler.List)
		playlistGroup.GET("/search", playlistHandler.Search)
		playlistGroup.GET("/batch-get", playlistHandler.BatchGet)
		playlistGroup.POST("/videos/update-order", playlistHandler.VideosUpdateOrder)
		playlistGroup.GET("/videos/order", playlistHandler.VideosOrder)
		playlistGroup.POST("/change/access/type", playlistHandler.ChangeAccessType)
		playlistGroup.POST("/i18n/create", playlistHandler.TranslatePlaylist)
		playlistGroup.GET("/i18n", playlistHandler.GetPlaylistI18n)
		playlistGroup.POST("/i18n/batch-modify", playlistHandler.BatchModifyI18n)
	}
}

func registerAdsRoutes(apiGroup *gin.RouterGroup, adsHandler *handler.AdsHandler, jwt *jwt.JWT, logger *log.Logger) {
	adsRouter := apiGroup.Group("/ads")
	{
		// Unauthenticated route requiring SiteID  && to  client
		adsRouter.Group("").Use(middleware.RequireSiteID(logger)).GET("/unit/conf", adsHandler.UnitConf)

		// Authenticated routes
		authedAdsGroup := adsRouter.Group("").Use(middleware.StrictAuth(jwt, logger))
		{
			authedAdsGroup.POST("/unit/create", adsHandler.CreateAdUnit)
			authedAdsGroup.GET("/unit/list", adsHandler.UnitList)
			authedAdsGroup.POST("/unit/modify", adsHandler.UnitModify)
		}
	}
}

func registerPaymentRoutes(apiGroup *gin.RouterGroup, paymentHandler *handler.PaymentHandler, coinsHandler *handler.CoinsHandler, subscriptionHandler *handler.SubscriptionHandler, jwt *jwt.JWT, logger *log.Logger) {
	// Creator-side payment routes
	paymentGroup := apiGroup.Group("/payment").Use(middleware.StrictAuth(jwt, logger))
	{
		//Test payment configuration
		paymentGroup.POST("/conf/test", paymentHandler.ConfTest)
		//Save payment configuration
		paymentGroup.POST("/conf/save", paymentHandler.ConfSave)
		//Get payment configuration information
		paymentGroup.GET("/conf/info", paymentHandler.ConfInfo)

		//Get account information
		paymentGroup.GET("/account/info", paymentHandler.AccountInfo)
		//Create coin package
		paymentGroup.POST("/coins/package/create", paymentHandler.CreateCoinPackage)
		// List coin packages
		paymentGroup.GET("/coins/package/list", coinsHandler.ListCoinPackage)
		paymentGroup.POST("/coins/package/modify", coinsHandler.ModifyCoinPackage)

		// Subscription management routes
		paymentGroup.POST("/subscription/create", subscriptionHandler.CreateSubscription)
		paymentGroup.POST("/subscription/modify", subscriptionHandler.ModifySubscription)
		paymentGroup.GET("/subscription/list", subscriptionHandler.ListSubscriptions)
		paymentGroup.GET("/subscription/get", subscriptionHandler.GetSubscription)

		// Creator manually grants coins to a user.
		paymentGroup.POST("/customers/coins/grant", coinsHandler.GrantCoinsToCustomer)
		paymentGroup.GET("/customers/coins/transactions", paymentHandler.CustomerTransactionHistory)
		paymentGroup.GET("/customers/coins/videos/transactions", paymentHandler.CustomerVideoUnlockHistory)
		paymentGroup.GET("/customers/coins/balance", coinsHandler.GetCustomerCoinsBalance)
		paymentGroup.POST("/customers/subscription/cancel", subscriptionHandler.CancelCustomerSubscription)
	}
	// Payment callback interface (no auth)
	paymentNoAuthGroup := apiGroup.Group("/payment")
	{
		paymentNoAuthGroup.POST("callback/stripe", paymentHandler.StripeCallback)
		paymentNoAuthGroup.GET("callback/stripe/fulfill", paymentHandler.StripeFulfillOrder)
		paymentNoAuthGroup.POST("callback/stripe/fulfill", paymentHandler.StripeFulfillOrder)
		paymentNoAuthGroup.POST("callback/paypal", paymentHandler.PayPalCallback)
	}
}

func registerAnalyticsRoutes(apiGroup *gin.RouterGroup, analyticsHandler *handler.AnalyticsHandler, jwt *jwt.JWT, logger *log.Logger) {
	analyticsGroup := apiGroup.Group("/analytics").Use(middleware.StrictAuth(jwt, logger))
	{
		//All transaction records for the current site
		analyticsGroup.POST("/income/transactions", analyticsHandler.IncomeTransactions)
		//Some transaction record's detailed information
		analyticsGroup.GET("/income/transactions/info", analyticsHandler.IncomeTransactionsInfo)
		//Income statistics by day, returns daily income statistics.
		analyticsGroup.POST("/income/statistics", analyticsHandler.IncomStatistics)
	}
}

func registerUserRoutes(apiGroup *gin.RouterGroup, userHandler *handler.UserHandler, jwt *jwt.JWT, logger *log.Logger) {
	// User endpoints with SiteMiddleware
	userGroup := apiGroup.Group("/user")
	userGroup.Use(middleware.RequireSiteID(logger))
	{
		userGroup.POST("/register", userHandler.Register)
		userGroup.POST("/login", userHandler.Login)
		userGroup.POST("/login/oauth2", userHandler.LoginByOauth2)

		// Routes that require authentication
		authUserGroup := userGroup.Group("")
		authUserGroup.Use(middleware.StrictAuth(jwt, logger)) // Applied to the sub-group
		{
			authUserGroup.GET("/profile", userHandler.GetProfile)
			authUserGroup.POST("/profile-modify", userHandler.ProfileModify) //修改用户信息
			authUserGroup.POST("/change-password", userHandler.ChangePassword)
			authUserGroup.POST("/meta-click/sync", userHandler.SyncMetaClick)
			authUserGroup.POST("/pixel/sync", userHandler.SyncPixel)
		}
	}
}

func registerClientRoutes(apiGroup *gin.RouterGroup, clientPlayerHandler *handler.ClientPlayerHandler,
	paymentHandler *handler.PaymentHandler, coinsHandler *handler.CoinsHandler, subscriptionHandler *handler.SubscriptionHandler, jwt *jwt.JWT, logger *log.Logger) {
	// Client API endpoints with SiteMiddleware and Auth
	// playerGroup := apiGroup.Group("/client").Use(middleware.RequireSiteID(logger)).Use(middleware.StrictAuth(jwt, logger))
	// {
	// 	playerGroup.GET("/playlist/info", clientPlayerHandler.PlaylistInfo)
	// 	playerGroup.GET("/playlist/videos", clientPlayerHandler.PlaylistVideos)
	// 	playerGroup.GET("/playlist/related", clientPlayerHandler.PlaylistRelatedRecommend)
	// }

	playerOptionalAuthGroup := apiGroup.Group("/client").Use(middleware.RequireSiteID(logger)).Use(middleware.OptionalAuth(jwt, logger)).Use(middleware.RequireI18n(logger))
	{

		playerOptionalAuthGroup.GET("/feed", clientPlayerHandler.Feed)
		// playerOptionalAuthGroup.GET("/playlist/recommend", clientPlayerHandler.PlaylistRecommend) // Recommend playlists
		playerOptionalAuthGroup.GET("/playlist/search", clientPlayerHandler.PlaylistSearch)
		playerOptionalAuthGroup.GET("/playlist/info", clientPlayerHandler.PlaylistInfo)
		playerOptionalAuthGroup.GET("/playlist/batch-get", clientPlayerHandler.BatchPlaylistInfo)
		playerOptionalAuthGroup.GET("/playlist/videos", clientPlayerHandler.PlaylistVideos)
		playerOptionalAuthGroup.GET("/playlist/related", clientPlayerHandler.PlaylistRelatedRecommend)

	}

	// Client Site Info (No Auth, but could have SiteID if needed, or handled by handler)
	playerSite := apiGroup.Group("/client/site")
	{
		playerSite.GET("/info", clientPlayerHandler.SiteInfo)

		playerWithSiteID := playerSite.Use(middleware.RequireSiteID(logger))
		playerWithSiteID.GET("/pages", clientPlayerHandler.SitePages)

		playerWithSiteID.GET("/new-release", clientPlayerHandler.NewRelease)  // New release videos
		playerWithSiteID.GET("/playlists", clientPlayerHandler.AllPlaylistID) // Get all playlist IDs for the site
		playerWithSiteID.GET("/plist", clientPlayerHandler.SitePlist)         // Get published playlist IDs excluding m1 traffic

	}

	// Client Anonymous Registration (Requires SiteID)
	anonymousGroup := apiGroup.Group("/client/anonymous").Use(middleware.RequireSiteID(logger))
	{
		anonymousGroup.POST("/register", clientPlayerHandler.AnonRegister)
	}

	// Client file upload (No authentication required)
	playerUpload := apiGroup.Group("/client")
	{
		playerUpload.POST("/upload", clientPlayerHandler.UploadFile)
		playerUpload.POST("/transfer/save", clientPlayerHandler.TransferSave)
		playerUpload.POST("/transfer/get", clientPlayerHandler.TransferGet)
	}

	// Client-end payment related interfaces
	clientPaymentRouter := apiGroup.Group("/client/payment")
	clientPaymentRouter.Use(middleware.RequireSiteID(logger))
	{
		// Unauthenticated routes
		clientPaymentRouter.GET("/coins/package/list", coinsHandler.PkgClientList)
		clientPaymentRouter.GET("/subscription/package/list", subscriptionHandler.ListClientSubscriptions)
		clientPaymentRouter.GET("/provider/list", paymentHandler.ProviderList)

		// Authenticated routes
		authedClientPaymentGroup := clientPaymentRouter.Group("")
		authedClientPaymentGroup.Use(middleware.StrictAuth(jwt, logger))
		{
			authedClientPaymentGroup.POST("/order/create", paymentHandler.OrderCreate)
			authedClientPaymentGroup.GET("/purchases", paymentHandler.GetUserPurchases)
			authedClientPaymentGroup.POST("/coins/videos/buy", coinsHandler.BuyVideoWithCoins)
			authedClientPaymentGroup.GET("/coins/balance", coinsHandler.GetBalance)
			authedClientPaymentGroup.GET("/coins/transactions", coinsHandler.GetAddCoionsHistory)
			authedClientPaymentGroup.GET("/coins/videos/transactions", coinsHandler.GetVideoUnlockHistory)
			authedClientPaymentGroup.POST("/coins/claim-task", coinsHandler.ClaimTaskReward)
			authedClientPaymentGroup.GET("/coins/wheel-status", coinsHandler.GetWheelStatus)
			authedClientPaymentGroup.POST("/coins/wheel-spin", coinsHandler.SpinWheel)
			authedClientPaymentGroup.POST("/subscription/create", subscriptionHandler.SubscriptionCreate)
			authedClientPaymentGroup.POST("/subscription/confirm", subscriptionHandler.SubscriptionConfirm)
			authedClientPaymentGroup.GET("/subscription/user/list", subscriptionHandler.GetUserSubscriptions)
			authedClientPaymentGroup.POST("/subscription/cancel", subscriptionHandler.CancelSubscription)
		}
	}

	clientVideoActionsGroup := apiGroup.Group("/client/video")
	clientVideoActionsGroup.Use(middleware.RequireSiteID(logger))
	{
		authedClientVideoGroup := clientVideoActionsGroup.Group("").Use(middleware.StrictAuth(jwt, logger))
		{
			authedClientVideoGroup.POST("/playback/records", clientPlayerHandler.ReportRecords)
			authedClientVideoGroup.GET("/history", clientPlayerHandler.GetWatchHistory)
		}
	}
}

func registerPageBuilderRoutes(apiGroup *gin.RouterGroup, pagesBuilderHandler *handler.PagesBuilderHandler, jwt *jwt.JWT, logger *log.Logger) {
	// Page routes
	pageGroup := apiGroup.Group("/pages-builder").Use(middleware.StrictAuth(jwt, logger))
	{
		pageGroup.POST("/save", pagesBuilderHandler.SavePagesBuilderData)
		pageGroup.GET("/info", pagesBuilderHandler.GetPagesBuilderData)
		pageGroup.POST("/publish", pagesBuilderHandler.PublishPagesBuilderData)
		pageGroup.GET("/templates", pagesBuilderHandler.TemplatesList)
		pageGroup.POST("/translate", pagesBuilderHandler.TranslatePages)
		// pageGroup.GET("/publish/history", pagesBuilderHandler.GetPublishHistory)
	}
}

func registerA2ERoutes(apiGroup *gin.RouterGroup, a2eHandler *handler.A2EHandler, jwt *jwt.JWT, logger *log.Logger) {
	a2eGroup := apiGroup.Group("/a2e").Use(middleware.SiteMiddleware(logger)).Use(middleware.StrictAuth(jwt, logger))
	{
		a2eGroup.POST("/wan2_7/invoke", a2eHandler.InvokeWan27)
	}
}

func registerPromptRoutes(apiGroup *gin.RouterGroup, promptHandler *handler.PromptHandler, jwt *jwt.JWT, logger *log.Logger) {
	promptGroup := apiGroup.Group("/prompt").Use(middleware.StrictAuth(jwt, logger))
	{
		promptGroup.POST("/optimize", promptHandler.Optimize)
	}
}

func registerPluginsRoutes(apiGroup *gin.RouterGroup, pluginHandler *handler.PluginHandler, jwt *jwt.JWT, logger *log.Logger, conf *viper.Viper) {
	// 插件管理路由 - 不需要认证
	pluginGroup := apiGroup.Group("/plugins")
	{
		// 注册插件
		pluginGroup.POST("/register", pluginHandler.RegisterPlugin)
	}

	// 插件管理路由 - 需要管理员认证
	pluginManagementGroup := apiGroup.Group("/plugins").Use(middleware.StrictAuth(jwt, logger)).Use(middleware.SiteMiddleware(logger))
	{
		// 获取可用插件列表
		pluginManagementGroup.GET("/list", pluginHandler.ListPlugins)
		// 安装插件到站点
		pluginManagementGroup.POST("/install", pluginHandler.InstallPlugin)
		// 卸载站点插件
		pluginManagementGroup.POST("/uninstall", pluginHandler.UninstallPlugin)
		// 获取站点已安装插件列表
		pluginManagementGroup.GET("/installed/list", pluginHandler.ListInstalledPlugins)
	}

	// 插件调用路由 - 需要站点认证和插件认证
	pluginCallGroup := apiGroup.Group("/plugins").Use(middleware.StrictAuth(jwt, logger)).Use(middleware.SiteMiddleware(logger)).Use(middleware.PluginAuthMiddleware(conf, logger))
	{
		// 主动调用接口 - 需要插件认证（中间件只做加密验证，不查询数据库）
		pluginCallGroup.POST("/hook/call", pluginHandler.Call)
	}
}

// registerInternalCoinsRoutes registers internal service-to-service coin operation routes
// Requires JWT authentication and Site middleware
func registerInternalCoinsRoutes(apiGroup *gin.RouterGroup, coinsInternalHandler *handler.CoinsInternalHandler, jwt *jwt.JWT, logger *log.Logger) {
	internalGroup := apiGroup.Group("/internal/coins").Use(middleware.StrictAuth(jwt, logger)).Use(middleware.SiteMiddleware(logger))
	{
		// Get user coin balance
		internalGroup.GET("/balance", coinsInternalHandler.InternalGetBalance)
		// Add coins (treated as coin package purchase)
		internalGroup.POST("/add", coinsInternalHandler.InternalAddCoins)
		// Deduct coins
		internalGroup.POST("/deduct", coinsInternalHandler.InternalDeductCoins)
	}
}

// registerInternalVideoRoutes registers internal service-to-service video operation routes
// No authentication required
func registerInternalVideoRoutes(apiGroup *gin.RouterGroup, videoHandler *handler.VideoHandler) {
	internalGroup := apiGroup.Group("/internal/video")
	{
		// Get video config (no authentication required)
		internalGroup.GET("/config", videoHandler.InternalGetVideoConfig)
	}
}

func registerIapRoutes(apiGroup *gin.RouterGroup, iapHandler *handler.IapHandler, jwt *jwt.JWT, logger *log.Logger) {
	iapGroup := apiGroup.Group("/iap").Use(middleware.StrictAuth(jwt, logger))
	{
		//ios android
		iapGroup.POST("/verifysub", iapHandler.VerifySub)
		iapGroup.POST("/verify", iapHandler.Verify)
	}

	// Apple / Google server notifications (no user JWT)
	iapNoAuthGroup := apiGroup.Group("/iap")
	{
		iapNoAuthGroup.POST("/notify", iapHandler.Notify)
	}
}

// registerSitePageConfigRoutes registers site page config routes
func registerSitePageConfigRoutes(apiGroup *gin.RouterGroup, sitePageConfigHandler *handler.SitePageConfigHandler, jwt *jwt.JWT, logger *log.Logger) {
	configGroup := apiGroup.Group("/site/page-config").Use(middleware.SiteMiddleware(logger)).Use(middleware.StrictAuth(jwt, logger))
	{
		configGroup.POST("/create", sitePageConfigHandler.Create)
		configGroup.POST("/update", sitePageConfigHandler.Update)
		configGroup.GET("/get", sitePageConfigHandler.Get)
		configGroup.GET("/list", sitePageConfigHandler.List)
	}
}
