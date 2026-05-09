//go:build wireinject
// +build wireinject

package wire

import (
	adapter "shortpress-server/internal/adapter/payment/stripe"
	"shortpress-server/internal/handler"
	"shortpress-server/internal/plugins"
	"shortpress-server/internal/repository/db"
	"shortpress-server/internal/repository/db/ads"
	"shortpress-server/internal/repository/db/creator"
	"shortpress-server/internal/repository/db/payment"
	"shortpress-server/internal/repository/db/playlist"
	pluginsrepo "shortpress-server/internal/repository/db/plugins"
	"shortpress-server/internal/repository/db/site"
	"shortpress-server/internal/repository/db/user"
	"shortpress-server/internal/repository/db/video"
	"shortpress-server/internal/server"
	"shortpress-server/internal/service"
	"shortpress-server/internal/service/analytics"
	"shortpress-server/internal/service/cache"
	"shortpress-server/internal/service/payment/coins"
	paypalservice "shortpress-server/internal/service/payment/paypal"
	"shortpress-server/internal/service/payment/stripe"
	"shortpress-server/internal/service/payment/sub"

	"shortpress-server/pkg/app"
	"shortpress-server/pkg/blacklist"
	"shortpress-server/pkg/jwt"
	"shortpress-server/pkg/log"
	"shortpress-server/pkg/oauth2"
	"shortpress-server/pkg/server/http"
	"shortpress-server/pkg/sid"

	"github.com/google/wire"
	"github.com/spf13/viper"
)

// func NewCacheMiddleware(cacheService *cache.CacheService, logger *log.Logger) *middleware.CacheMiddleware {
//     return middleware.NewCacheMiddleware(*cacheService, logger)
// }

// var middlewareSet = wire.NewSet(
//     NewCacheMiddleware,
// )

var repositorySet = wire.NewSet(
	db.NewDB,
	// db.NewRedis,
	db.NewRepository,
	db.NewTransaction,
	creator.NewCreatorRepository,
	site.NewSiteRepository,
	site.NewSiteBuilderDataRepository,
	site.NewSiteCurrentPublishedRepository,
	site.NewSitePublishedHistoryRepository,
	site.NewSitePageTemplateRepository,
	site.NewSitePageConfigRepository,
	video.NewVideoRepository,
	video.NewVideoSeoRepository,
	video.NewVideoSourceRepository,
	playlist.NewPlaylistRepository,
	creator.NewCreatorProfileRepository,
	site.NewSiteSeoRepository,
	creator.NewCreatorWebsiteRepository,
	creator.NewCreatorGuidesRepository,
	site.NewSitePlaylistRepository,
	playlist.NewPlaylistVidRepository,
	playlist.NewPlaylistSeoRepository,
	playlist.NewPlaylistI18nRepository,
	ads.NewAdRepository,
	ads.NewAdLocationRepository,
	user.NewUserRepository,
	user.NewUserAuthRepository,
	user.NewUserProfileRepository,
	user.NewUserPlayRecordsRepository,
	payment.NewPaymentConfgRepository,
	payment.NewCoinPackageRepository,
	payment.NewPaymentTransactionRepository,
	payment.NewCoinTransactionRepository,
	payment.NewUserCoinsRepository,
	payment.NewContentUnlockRepository,
	payment.NewWebhookEventRepository,
	payment.NewSubscriptionPackageRepository,
	payment.NewUserSubscriptionRepository,
	pluginsrepo.NewPluginOrderRepository,
	pluginsrepo.NewPluginRepository,
	pluginsrepo.NewSitePluginRepository,
	// redis.NewRedisRepository,
)

var serviceSet = wire.NewSet(
	service.NewService,
	service.NewCreatorService,
	service.NewSiteService,
	service.NewSitePageConfigService,
	service.NewVideoService,
	service.NewPlaylistService,
	service.NewClientPlayerService,
	service.NewAdsService,
	service.NewUserService,
	service.NewPagesBuilderService,
	service.NewA2EService,
	service.NewPromptService,
	service.NewPaymentBiz,
	adapter.NewClient,
	stripe.NewStripeService,
	paypalservice.NewPaypalServiceFull,
	coins.NewCoinsService,
	service.NewAnalyticsService,
	analytics.NewTrackingService,
	sub.NewSubscriptionService,
	cache.NewCacheService,
	cache.NewContentCacheService,
	service.NewPluginService,
)

// 插件系统
var pluginsSet = wire.NewSet(
	plugins.NewPlugins,
)

var handlerSet = wire.NewSet(
	handler.NewHandler,
	handler.NewCreatorHandler,
	handler.NewSiteHandler,
	handler.NewSitePageConfigHandler,
	handler.NewVideoHandler,
	handler.NewPlaylistHandler,
	handler.NewClientPlayerHandler,
	handler.NewAdsHandler,
	handler.NewUserHandler,
	handler.NewPaymentHandler,
	handler.NewCoinsHandler,
	handler.NewCoinsInternalHandler,
	handler.NewAnalyticsHandler,
	handler.NewSubscriptionHandler,
	handler.NewPagesBuilderHandler,
	handler.NewA2EHandler,
	handler.NewPromptHandler,
	handler.NewPluginHandler,
	handler.NewIapHandler,
)

var serverSet = wire.NewSet(
	server.NewHTTPServer,
)

func newApp(
	httpServer *http.Server,
) *app.App {
	return app.NewApp(
		app.WithServer(httpServer),
		app.WithName("shortpress-server"),
	)
}

// ProvideOauthTypes provides the oauth types for the oauth2 client
func ProvideOauthTypes() []oauth2.OauthType {
	return []oauth2.OauthType{
		oauth2.TypeGoogle,
		oauth2.TypeTikTok,
	}
}

func NewWire(*viper.Viper, *log.Logger) (*app.App, func(), error) {
	panic(wire.Build(
		repositorySet,
		serviceSet,
		pluginsSet,
		handlerSet,
		// middlewareSet,  // 添加中间件集合
		serverSet,
		sid.NewSid,
		jwt.NewJwt,
		ProvideOauthTypes,
		oauth2.New,
		blacklist.NewBlacklist, // 添加黑名单
		newApp,
	))
}
