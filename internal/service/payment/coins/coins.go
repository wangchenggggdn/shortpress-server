package coins

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"shortpress-server/internal/api"
	"shortpress-server/internal/common"
	"shortpress-server/internal/model"
	"shortpress-server/internal/repository/db"
	"shortpress-server/internal/repository/db/creator"
	"shortpress-server/internal/repository/db/payment"
	"shortpress-server/internal/repository/db/playlist"
	"shortpress-server/internal/repository/db/user"
	"shortpress-server/internal/repository/db/video"
	"shortpress-server/internal/types"
	"shortpress-server/pkg/log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type CoinsService interface {
	// GetBalance gets a user's coin balance
	GetBalance(ctx context.Context, userID, siteID string) (*api.UserCoinsResponse, error)
	GetBalanceByEmail(ctx *gin.Context, email string, siteID string) (*api.UserCoinsResponse, error)

	AddCoinsByPayment(ctx context.Context, amount int, source string, transaction *model.PaymentTransaction) (*model.CoinTransaction, error)

	// Updated AddCoins with struct parameter
	AddCoins(ctx context.Context, params *CoinAdditionParams) (*model.CoinTransaction, error)

	// GetTransactionHistory gets a user's coin transaction history
	GetTransactionHistory(ctx context.Context, userID, siteID string, page, pageSize int) (*api.CoinTransactionHistoryResponse, error)

	// ListPackages gets all available coin packages for a site
	ListPackages(ctx context.Context, siteID string, status int) ([]*api.CoinPackageResponseData, error)

	// GetContentUnlockHistory gets a user's content unlock history
	GetContentUnlockHistory(ctx context.Context, userID, siteID string, page, pageSize int) (*api.VideoUnlockHistoryResponse, error)

	// BuyContentWithCoins purchases content (video or playlist) with coins
	BuyContentWithCoins(ctx *gin.Context, userID, siteID string, req api.BuyContentWithCoinsRequest) (*api.BuyContentWithCoinsResponse, error)

	// GrantCoins allows creators to grant coins to a user identified by email
	GrantCoins(ctx *gin.Context, userEmail, siteID string, amount int, reason, creatorID string) (*api.GrantCoinsResponse, error)

	// UpdateCoinPackage updates an existing coin package
	UpdateCoinPackage(ctx context.Context, coinPackage *model.CoinPackage) error

	// Internal APIs for service-to-service communication
	InternalGetBalance(ctx *gin.Context, userID, siteID string) (*api.InternalGetBalanceResponse, error)
	InternalAddCoins(ctx *gin.Context, userID, siteID string, req api.InternalAddCoinsRequest) (*api.InternalAddCoinsResponse, error)
	InternalDeductCoins(ctx *gin.Context, userID, siteID string, req api.InternalDeductCoinsRequest) (*api.InternalDeductCoinsResponse, error)

	// ClaimTaskReward allows users to claim rewards for completing tasks
	ClaimTaskReward(ctx *gin.Context, userID, siteID, taskName string) (*api.ClaimTaskRewardResponse, error)

	// GetWheelStatus returns wheel availability and prize config
	GetWheelStatus(ctx *gin.Context, userID, siteID string, isVIP bool) (*api.WheelStatusResponse, error)

	// SpinWheel executes a free or paid wheel spin
	SpinWheel(ctx *gin.Context, userID, siteID string, isVIP bool, mode string) (*api.WheelSpinResponse, error)
}

type coinsService struct {
	logger                       *log.Logger
	tx                           db.Transaction
	userCoinsRepo                payment.UserCoinsRepository
	coinTransactionRepo          payment.CoinTransactionRepository
	coinPackageRepo              payment.CoinPackageRepository
	contentUnlockRepo            payment.ContentUnlockRepository
	playlistRepository           playlist.PlaylistRepository
	userRepository               user.UserRepository
	creatorRepository            creator.CreatorRepository
	videoRepository              video.VideoRepository
	paymentTransactionRepository payment.PaymentTransactionRepository
}

// Add this struct to group the parameters for AddCoins
type CoinAdditionParams struct {
	UserID         string
	SiteID         string
	Amount         int
	Source         string
	Description    string
	RelatedID      string
	RelatedType    int
	RealMoneySpent int64
	AdminID        string
	Snapshot       model.JSONMap
}

// Define a struct to group parameters for executePurchase method
type purchaseParams struct {
	UserID        string
	SiteID        string
	ContentID     string
	ContentTile   string
	PlaylistTitle string
	EpisodeNumber int
	ContentType   string
	PlaylistID    string
	CoinCost      int
}

// NewCoinsService creates a new coins service
func NewCoinsService(
	logger *log.Logger,
	tx db.Transaction,
	userCoinsRepo payment.UserCoinsRepository,
	coinTransactionRepo payment.CoinTransactionRepository,
	coinPackageRepo payment.CoinPackageRepository,
	contentUnlockRepo payment.ContentUnlockRepository,
	playlistRepository playlist.PlaylistRepository,
	userRepository user.UserRepository,
	creatorRepository creator.CreatorRepository,
	videoRepository video.VideoRepository,
	paymentTransactionRepository payment.PaymentTransactionRepository,

) CoinsService {
	return &coinsService{
		logger:                       logger,
		tx:                           tx,
		userCoinsRepo:                userCoinsRepo,
		coinTransactionRepo:          coinTransactionRepo,
		coinPackageRepo:              coinPackageRepo,
		contentUnlockRepo:            contentUnlockRepo,
		playlistRepository:           playlistRepository,
		userRepository:               userRepository,
		creatorRepository:            creatorRepository,
		videoRepository:              videoRepository,
		paymentTransactionRepository: paymentTransactionRepository,
	}
}

func (s *coinsService) Logger() *log.Logger {
	return s.logger
}

func (s *coinsService) Tx() db.Transaction {
	return s.tx
}

// GetBalance gets a user's coin balance
func (s *coinsService) GetBalance(ctx context.Context, userID, siteID string) (*api.UserCoinsResponse, error) {
	userCoins, err := s.userCoinsRepo.GetByUserAndSite(ctx, userID, siteID)
	if err != nil {
		return nil, err
	}

	response := &api.UserCoinsResponse{
		Balance: 0,
	}

	totalSpent, err := s.paymentTransactionRepository.GetUserTotalAmount(ctx, userID, siteID, time.Time{}, time.Time{})
	if err != nil {
		return nil, err
	}
	if userCoins != nil {
		// Total available balance is present + balance
		response.Balance = userCoins.Present + userCoins.Balance
		response.TotalEarned = userCoins.TotalEarned
		response.TotalSpent = userCoins.TotalSpent
		// response.TotalRealMoneySpent = types.FromCents(userCoins.TotalRealMoneySpent)
		response.TotalRealMoneySpent = types.FromCents(totalSpent)
	}

	return response, nil
}

func (s *coinsService) AddCoinsByPayment(ctx context.Context, amount int, source string,
	paymentTran *model.PaymentTransaction) (*model.CoinTransaction, error) {
	// Check if the payment transaction is valid
	if paymentTran == nil {
		return nil, errors.New("payment transaction is nil")
	}

	return s.AddCoins(ctx, &CoinAdditionParams{
		UserID: paymentTran.UserID,
		SiteID: paymentTran.SiteID,
		Amount: amount,
		// Source:      source,
		Source:         "purchase",
		Description:    "purchase",
		RelatedID:      paymentTran.TransactionID,
		RelatedType:    model.CoinRelatedTypePayment,
		RealMoneySpent: paymentTran.Amount,
		Snapshot: map[string]interface{}{
			"payment_transcation_id":           paymentTran.TransactionID,
			"payment_transcation_provider":     paymentTran.Provider,
			"payment_transcation_status":       paymentTran.Status,
			"payment_transcation_amount":       paymentTran.Amount,
			"payment_transcation_currency":     paymentTran.Currency,
			"payment_transcation_created_at":   paymentTran.CreatedAt.Unix(),
			"payment_transcation_payment_type": paymentTran.PaymentType,
		},
	})
}

// Implement the new method
func (s *coinsService) AddCoins(ctx context.Context, params *CoinAdditionParams) (*model.CoinTransaction, error) {
	if params == nil || params.Amount <= 0 {
		return nil, errors.New("amount must be positive")
	}

	var transaction *model.CoinTransaction
	//事务在最外面
	// err := s.Tx().Transaction(ctx, func(ctx context.Context) error {
	// Update user balance
	userCoins, err := s.userCoinsRepo.UpdateBalance(ctx, params.UserID, params.SiteID, params.Amount, params.RealMoneySpent)
	if err != nil {
		return nil, err
	}

	// Create transaction record
	transactionID := uuid.New().String()
	coinTx := &model.CoinTransaction{
		TransactionID: transactionID,
		UserID:        params.UserID,
		SiteID:        params.SiteID,
		Amount:        params.Amount,
		BeforeBalance: userCoins.Balance - params.Amount,
		Balance:       userCoins.Balance,
		Source:        params.Source,
		RelatedID:     params.RelatedID,
		RelatedType:   params.RelatedType,
		AdminID:       params.AdminID,
		Description:   params.Description,
		Snapshot:      params.Snapshot,
	}

	err = s.coinTransactionRepo.Create(ctx, coinTx)
	if err != nil {
		return nil, fmt.Errorf("failed to create coin transaction: %w", err)
	}

	transaction = coinTx
	// return nil
	// })

	if err != nil {
		return nil, err
	}

	return transaction, nil
}

// GetTransactionHistory gets a user's coin transaction history
func (s *coinsService) GetTransactionHistory(ctx context.Context, userID, siteID string, page, pageSize int) (*api.CoinTransactionHistoryResponse, error) {
	offset := (page - 1) * pageSize

	transactions, err := s.coinTransactionRepo.ListAddCoionsByUserID(ctx, userID, pageSize, offset)
	if err != nil {
		return nil, err
	}

	count, err := s.coinTransactionRepo.CountByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Convert model transactions to API response format
	responseTransactions := make([]*api.CoinTransactionResponse, 0, len(transactions))
	for _, tx := range transactions {
		responseTransactions = append(responseTransactions, &api.CoinTransactionResponse{
			TransactionID: tx.TransactionID,
			Amount:        tx.Amount,
			BeforeBalance: tx.BeforeBalance,
			Balance:       tx.Balance,
			Source:        tx.Source,
			RelatedType:   tx.RelatedType,
			Description:   tx.Description,
			CreatedAt:     tx.CreatedAt.Unix(),
		})
	}

	return &api.CoinTransactionHistoryResponse{
		Items:    responseTransactions,
		Total:    count,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// ListPackages gets all available coin packages for a site
func (s *coinsService) ListPackages(ctx context.Context, siteID string, status int) ([]*api.CoinPackageResponseData, error) {
	// Get the coin packages from repository
	packages, err := s.coinPackageRepo.ListBySiteID(ctx, siteID, status)
	if err != nil {
		return nil, err
	}

	// Convert to API response format
	var result []*api.CoinPackageResponseData
	for _, pkg := range packages {
		result = append(result, &api.CoinPackageResponseData{
			PackageID:          pkg.PackageID,
			Name:               pkg.Name,
			Description:        pkg.Description,
			Features:           pkg.Features,
			CoinAmount:         pkg.CoinAmount,
			Price:              types.FromCents(pkg.Price),
			OriginalPrice:      types.FromCents(pkg.OriginalPrice),
			Currency:           pkg.Currency,
			DiscountPercentage: pkg.DiscountPercentage,
			Status:             pkg.Status,
			IOSProductID:       pkg.IOSProductID,
		})
	}

	return result, nil
}

// GetContentUnlockHistory gets a user's content unlock history
func (s *coinsService) GetContentUnlockHistory(ctx context.Context, userID, siteID string, page, pageSize int) (*api.VideoUnlockHistoryResponse, error) {
	offset := (page - 1) * pageSize

	unlocks, err := s.contentUnlockRepo.ListByUserID(ctx, userID, pageSize, offset)
	if err != nil {
		return nil, err
	}

	count, err := s.contentUnlockRepo.CountByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Convert model unlocks to API response format
	responseUnlocks := make([]*api.VideoUnlockResponse, 0, len(unlocks))
	for _, unlock := range unlocks {
		resp := &api.VideoUnlockResponse{
			ContentID:     unlock.ContentID,
			ContentType:   unlock.ContentType,
			ContentTitle:  unlock.ContentTitle,
			PlaylistTitle: unlock.PlaylistTitle,
			EpisodeNumber: unlock.EpisodeNumber,
			PlaylistID:    unlock.PlaylistID,
			CoinCost:      unlock.CoinCost,
			TransactionID: unlock.TransactionID,
			UnlockedAt:    unlock.CreatedAt.Unix(),
		}

		if unlock.ExpiredAt != nil {
			resp.ExpiredAt = unlock.ExpiredAt.Unix()
		}

		responseUnlocks = append(responseUnlocks, resp)
	}

	return &api.VideoUnlockHistoryResponse{
		Items:    responseUnlocks,
		Total:    count,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// BuyContentWithCoins purchases content (video or playlist) with coins
func (s *coinsService) BuyContentWithCoins(ctx *gin.Context, userID, siteID string, req api.BuyContentWithCoinsRequest) (*api.BuyContentWithCoinsResponse, error) {
	// Check if the content is already unlocked
	existingUnlock, err := s.contentUnlockRepo.CheckExistingUnlock(ctx, userID, req.ContentID, req.ContentType, req.PlaylistID)
	if err != nil {
		s.Logger().Error("Failed to check existing unlock", zap.Error(err))
		return nil, err
	}

	if existingUnlock != nil {
		// Content already unlocked, return success with existing unlock info
		return &api.BuyContentWithCoinsResponse{
			TransactionID: existingUnlock.TransactionID,
			CoinCost:      0, // No additional cost
			Balance:       s.getUserBalance(ctx, userID, siteID),
		}, nil
	}

	user, err := s.userRepository.GetByUserID(ctx, userID)
	if err != nil || user == nil {
		return nil, common.ErrUserNotFound
	}

	video, err := s.videoRepository.GetByVID(ctx, req.ContentID) //目前都是视频
	if err != nil {
		return nil, err
	}

	playlistInfo, err := s.playlistRepository.GetByPlaylistID(ctx, req.PlaylistID)
	if err != nil {
		s.Logger().Error("Failed to get playlist info", zap.Error(err))
		return nil, err
	}
	if playlistInfo == nil {
		log.Error(ctx, fmt.Sprintf("playlist with ID %s not found", req.PlaylistID))
		return nil, fmt.Errorf("playlist with ID %s not found", req.PlaylistID)
	}

	ep, err := s.getPlaylistEpisodeInfo(req.ContentID, playlistInfo)
	if err != nil {
		log.Warning(ctx, fmt.Sprintf("Failed to get episode info for video %s in playlist %s: %v", req.ContentID, req.PlaylistID, err))
	}

	// Determine the cost based on content type and pricing model
	coinCost, err := s.calculateContentCost(req.ContentID, req.ContentType, playlistInfo)
	if err != nil {
		return nil, err
	}

	// If free content, just create the unlock record without a transaction
	if coinCost == 0 {
		return nil, fmt.Errorf("content is free, no coins required")
	}

	// Execute the purchase transaction with the new struct
	params := &purchaseParams{
		UserID:        userID,
		SiteID:        siteID,
		ContentID:     req.ContentID,
		ContentType:   req.ContentType,
		ContentTile:   video.Title,
		PlaylistTitle: playlistInfo.Title,
		EpisodeNumber: ep,
		PlaylistID:    req.PlaylistID,
		CoinCost:      coinCost,
	}

	return s.executePurchase(ctx, params)
}

// Helper method to calculate content cost - now takes playlist model directly
func (s *coinsService) calculateContentCost(contentID, contentType string, playlistInfo *model.Playlist) (int, error) {
	var coinCost int

	// if contentType == "video" {
	//     if playlistInfo.AccessType == 1 || playlistInfo.AccessType == 0 {
	//         // Free access
	//         coinCost = 0
	//     } else {
	//         // Paid access
	//         coinCost = playlistInfo.SingleVideoPrice
	//     }
	// } else if contentType == "playlist" {
	//     return 0, fmt.Errorf("unsupported content type: %s", contentType)
	// }

	switch contentType {
	case "video":
		if playlistInfo.AccessType == 1 || playlistInfo.AccessType == 0 {
			// Free access
			coinCost = 0
		} else {
			// Paid access
			coinCost = playlistInfo.SingleVideoPrice
		}
	case "playlist":
		// Purchasing a whole playlist might have a different logic or might be unsupported
		// For now, it's explicitly unsupported as per original logic
		return 0, fmt.Errorf("unsupported content type for direct purchase: %s", contentType)
	default:
		// Handle unknown content types explicitly
		return 0, fmt.Errorf("unknown content type: %s", contentType)
	}

	return coinCost, nil
}

// Helper method to execute the purchase - now with struct parameter
func (s *coinsService) executePurchase(ctx context.Context, params *purchaseParams) (*api.BuyContentWithCoinsResponse, error) {
	var response *api.BuyContentWithCoinsResponse

	// Run everything in a transaction to ensure atomicity
	err := s.Tx().Transaction(ctx, func(ctx context.Context) error {
		// Check balance (present + balance)
		userCoins, err := s.userCoinsRepo.GetByUserAndSite(ctx, params.UserID, params.SiteID)
		if err != nil {
			return err
		}

		if userCoins == nil || (userCoins.Present+userCoins.Balance) < params.CoinCost {
			return common.ErrInsufficientCoins
		}

		// Get before balance for transaction record
		beforeBalance := userCoins.Present + userCoins.Balance

		// Deduct coins (prioritizing present first)
		userCoins, err = s.userCoinsRepo.DeductCoins(ctx, params.UserID, params.SiteID, params.CoinCost)
		if err != nil {
			return err
		}

		// Create transaction record
		transactionID := uuid.New().String()
		description := fmt.Sprintf("Purchase %s: %s", params.ContentType, params.ContentID)

		coinTx := &model.CoinTransaction{
			TransactionID: transactionID,
			UserID:        params.UserID,
			SiteID:        params.SiteID,
			Amount:        -params.CoinCost,
			BeforeBalance: beforeBalance,
			Balance:       userCoins.Present + userCoins.Balance,
			Source:        model.CoinSourceUnlock,
			RelatedID:     params.ContentID,
			RelatedType:   getRelatedTypeFromContentType(params.ContentType),
			Description:   description,
			Snapshot: model.JSONMap{
				"content_id":   params.ContentID,
				"content_type": params.ContentType,
				"playlist_id":  params.PlaylistID,
			},
		}

		if err := s.coinTransactionRepo.Create(ctx, coinTx); err != nil {
			return err
		}

		// Create unlock record using repository method
		err = s.contentUnlockRepo.Create(ctx, &model.ContentUnlock{
			UserID:        params.UserID,
			ContentID:     params.ContentID,
			ContentType:   params.ContentType,
			ContentTitle:  params.ContentTile,
			PlaylistTitle: params.PlaylistTitle,
			EpisodeNumber: params.EpisodeNumber,
			PlaylistID:    params.PlaylistID,
			TransactionID: transactionID,
			CoinCost:      params.CoinCost,
		})

		if err != nil {
			return err
		}

		response = &api.BuyContentWithCoinsResponse{
			TransactionID: transactionID,
			CoinCost:      params.CoinCost,
			Balance:       userCoins.Present + userCoins.Balance,
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return response, nil
}

// Helper method to get user's current balance (present + balance)
func (s *coinsService) getUserBalance(ctx context.Context, userID, siteID string) int {
	userCoins, err := s.userCoinsRepo.GetByUserAndSite(ctx, userID, siteID)
	if err != nil || userCoins == nil {
		return 0
	}
	return userCoins.Present + userCoins.Balance
}

// Helper function to determine related type from content type
func getRelatedTypeFromContentType(contentType string) int {
	switch contentType {
	case "video":
		return model.CoinRelatedTypeVideo
	case "playlist":
		return model.CoinRelatedTypePlaylist
	default:
		return 0
	}
}

// GrantCoins adds coins to a user's account based on their email address
func (s *coinsService) GrantCoins(ctx *gin.Context, userEmail, siteID string, amount int, reason, creatorID string) (*api.GrantCoinsResponse, error) {
	if amount <= 0 {
		return nil, errors.New("amount must be positive")
	}

	// Look up the user by email and siteID
	user, err := s.userRepository.GetByEmailAndSiteID(ctx, userEmail, siteID)
	if err != nil {
		log.Error(ctx, "Failed to get user by email: "+err.Error())
		return nil, err
	}

	if user == nil {
		return nil, common.ErrUserNotFound
	}

	//查询siteID的邮箱
	creator, err := s.creatorRepository.GetByCreatorID(ctx, creatorID)
	if err != nil {
		return nil, err
	}

	// Create a description if none was provided
	description := reason
	if description == "" {
		description = fmt.Sprintf("grant  %v coins to %s", amount, userEmail)
	}

	// Use AddCoins to add the coins to the user's account
	snapshot := model.JSONMap{
		"granted_by": creator.Email,
		"reason":     reason,
	}

	coinAdditionParams := &CoinAdditionParams{
		UserID:      user.UserID,
		SiteID:      siteID,
		Amount:      amount,
		Source:      model.CoinSourceAdminAdjustment,
		Description: description,
		RelatedID:   creatorID,
		RelatedType: model.CoinRelatedTypeAdminID,
		AdminID:     creatorID,
		Snapshot:    snapshot,
	}
	transaction, err := s.AddCoins(ctx, coinAdditionParams)
	if err != nil {
		log.Error(ctx, "Failed to add coins: "+err.Error())
		return nil, err
	}

	// Create a payment transaction record
	paymentSnapshot := snapshot
	paymentSnapshot["name"] = fmt.Sprintf("refill coins: %d", amount)
	paymentUUID := uuid.NewString()
	paymentTransaction := &model.PaymentTransaction{
		TransactionID:     paymentUUID,
		OrderID:           paymentUUID,
		UserID:            user.UserID,
		SiteID:            siteID,
		Amount:            0,
		Currency:          "USD",
		Provider:          "manual",
		ProviderPaymentID: "",
		PaymentType:       model.PaymentTypeGrantByAdmin,
		Status:            model.PaymentStatusSuccess,
		RelatedID:         "",
		RelatedType:       model.RelatedTypeGrantByAdmin,
		Snapshot:          paymentSnapshot, //同金币流水线记录
	}
	err = s.paymentTransactionRepository.Create(ctx, paymentTransaction)
	if err != nil {
		log.Warning(ctx, "Failed to create payment transaction, but continue: "+err.Error())
	}

	return &api.GrantCoinsResponse{
		UserEmail:      userEmail,
		AmountAdded:    amount,
		CurrentBalance: transaction.Balance,
		TransactionID:  transaction.TransactionID,
	}, nil
}

// GetPlaylistEpisodeInfo retrieves the episode number for a video within a playlist
// Now accepts the playlist info directly instead of fetching it again
func (s *coinsService) getPlaylistEpisodeInfo(vid string, playlistInfo *model.Playlist) (int, error) {
	if playlistInfo == nil {
		return 0, fmt.Errorf("playlist info is nil")
	}

	if playlistInfo.OrderVids == "" {
		return 0, nil
	}

	orderData := &api.VideoSortData{}
	err := json.Unmarshal([]byte(playlistInfo.OrderVids), orderData)
	if err != nil {
		return 0, nil
	}

	// Find the episode number for the specific video
	for i, v := range orderData.VIDs {
		if v == vid {
			return i + 1, nil
		}
	}

	return 0, fmt.Errorf("video with ID %s not found in playlist %s", vid, playlistInfo.PlaylistID)
}

// GetUserCoinsByEmail retrieves the user's coin balance by their email address
func (s *coinsService) GetBalanceByEmail(ctx *gin.Context, email string, siteID string) (*api.UserCoinsResponse, error) {
	if email == "" {
		return nil, fmt.Errorf("email is required")
	}
	user, err := s.userRepository.GetByEmailAndSiteID(ctx, email, siteID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		log.Error(ctx, fmt.Sprintf("user with email %s not found for site %s", email, siteID))
		return nil, common.ErrUserNotFound
	}
	return s.GetBalance(ctx, user.UserID, siteID)
}

// UpdateCoinPackage updates an existing coin package
func (s *coinsService) UpdateCoinPackage(ctx context.Context, coinPackage *model.CoinPackage) error {
	// First, check if the coin package exists
	existingPackage, err := s.coinPackageRepo.GetByPackageID(ctx, coinPackage.PackageID)
	if err != nil {
		return err
	}

	if existingPackage == nil {
		return common.ErrNotFound
	}

	// Check if the site ID matches
	if existingPackage.SiteID != coinPackage.SiteID {
		return common.ErrBadRequest
	}

	if coinPackage.Name != "" {
		existingPackage.Name = coinPackage.Name
	}
	if coinPackage.Description != "" {
		existingPackage.Description = coinPackage.Description
	}
	if coinPackage.Features != nil {
		existingPackage.Features = coinPackage.Features
	}
	if coinPackage.Status != 0 {
		existingPackage.Status = coinPackage.Status
	}
	existingPackage.IOSProductID = coinPackage.IOSProductID

	err = s.coinPackageRepo.Update(ctx, existingPackage)
	if err != nil {
		return err
	}

	return nil
}

// validateCoinPackage 验证充值金额是否在 package 范围内
// 只对支付充值（CoinSourcePurchase）进行验证
func (s *coinsService) validateCoinPackage(ctx context.Context, siteID string, amount int) error {
	// 1. 查询该站点所有启用的充值套餐
	packages, err := s.coinPackageRepo.ListBySiteID(ctx, siteID, 1) // status=1 表示启用
	if err != nil {
		return fmt.Errorf("failed to query coin packages: %w", err)
	}

	// 2. 如果没有任何 package，则不允许充值
	if len(packages) == 0 {
		return errors.New("no coin packages available for this site")
	}

	// 3. 检查金额是否匹配任何一个启用的 package
	for _, pkg := range packages {
		if pkg.CoinAmount == amount {
			// 找到匹配的 package，验证通过
			return nil
		}
	}

	// 4. 没有找到匹配的 package，返回错误
	// 构建友好的错误信息，列出可用的充值金额
	availableAmounts := make([]int, 0, len(packages))
	for _, pkg := range packages {
		availableAmounts = append(availableAmounts, pkg.CoinAmount)
	}

	return fmt.Errorf("coin amount %d does not match any available package. available amounts: %v", amount, availableAmounts)
}
