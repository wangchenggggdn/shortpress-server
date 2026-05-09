package service

import (
	"context"
	"encoding/json"
	"shortpress-server/internal/types"
	"shortpress-server/pkg/translate"
	"strings"
	"time"

	"shortpress-server/internal/api"
	"shortpress-server/internal/common"
	"shortpress-server/internal/model"
	"shortpress-server/internal/repository/db/site"
	"shortpress-server/pkg/log"

	"github.com/gin-gonic/gin"
)

type PagesBuilderService interface {
	SavePagesBuilderData(ctx *gin.Context, creatorID string, req *api.SavePagesBuilderDataRequest) (*api.SavePagesBuilderDataResponse, error)
	GetPagesBuilderData(ctx *gin.Context, creatorID string, siteID string) (*api.GetPagesBuilderDataResponse, error)
	PublishPagesBuilderData(ctx *gin.Context, creatorID string, req *api.PublishPagesBuilderDataRequest) (*api.PublishPagesBuilderDataResponse, error)
	GetPublishHistory(ctx *gin.Context, creatorID string, req *api.GetPublishHistoryRequest) (*api.GetPublishHistoryResponse, error)
	GetSitePages(ctx *gin.Context, siteID string) (*api.GetSitePagesResponse, error)
	ListTemplates(ctx *gin.Context, page, pageSize int) (*api.TemplateListResponseData, error)
	TranslatePages(ctx *gin.Context, req *api.PageTranslateRequest) ([]*api.PageTranslateResponse, error)
}

func NewPagesBuilderService(
	service *Service,
	siteBuilderDataRepository site.SiteBuilderDataRepository,
	siteCurrentPublishedRepository site.SiteCurrentPublishedRepository,
	sitePublishedHistoryRepository site.SitePublishedHistoryRepository,
	siteRepository site.SiteRepository,
	sitePageTemplateRepository site.SitePageTemplateRepository,
) PagesBuilderService {
	llm := types.GetLLMConfig()
	targetLang := types.GetTranslatorLang()
	return &pagesBuilderService{
		Service:                        service,
		siteBuilderDataRepository:      siteBuilderDataRepository,
		siteCurrentPublishedRepository: siteCurrentPublishedRepository,
		sitePublishedHistoryRepository: sitePublishedHistoryRepository,
		siteRepository:                 siteRepository,
		sitePageTemplateRepository:     sitePageTemplateRepository,
		translator:                     translate.NewTranslator(llm.BaseURL, llm.Model, llm.APIKey, targetLang),
	}
}

type pagesBuilderService struct {
	*Service
	siteBuilderDataRepository      site.SiteBuilderDataRepository
	siteCurrentPublishedRepository site.SiteCurrentPublishedRepository
	sitePublishedHistoryRepository site.SitePublishedHistoryRepository
	siteRepository                 site.SiteRepository
	sitePageTemplateRepository     site.SitePageTemplateRepository
	translator                     translate.Translator
}

func (s *pagesBuilderService) SavePagesBuilderData(ctx *gin.Context, creatorID string, req *api.SavePagesBuilderDataRequest) (*api.SavePagesBuilderDataResponse, error) {
	// Validate site ID
	if req.SiteID == "" {
		log.Error(ctx, common.ErrSiteNotFound)
		return nil, common.ErrSiteNotFound
	}
	exists, err := s.siteRepository.ExistsCreatorAndSiteID(ctx, creatorID, req.SiteID)
	if err != nil {
		log.Error(ctx, err)
		return nil, err
	}
	if !exists {
		log.Error(ctx, common.ErrSiteNotFound)
		return nil, common.ErrSiteNotFound
	}

	// Update or create site builder data
	savedData, err := s.siteBuilderDataRepository.UpdateBySiteID(ctx, req.SiteID, creatorID, req.SiteData)
	if err != nil {
		log.Error(ctx, err)
		return nil, err
	}

	response := &api.SavePagesBuilderDataResponse{
		SiteID:        savedData.SiteID,
		VersionNumber: savedData.VersionNumber,
		UpdatedAt:     savedData.UpdatedAt.Unix(),
	}

	log.AddNotice(ctx, "version_number", savedData.VersionNumber)
	log.AddNotice(ctx, "updated_at", savedData.UpdatedAt)
	return response, nil
}

func (s *pagesBuilderService) GetPagesBuilderData(ctx *gin.Context, creatorID string, siteID string) (*api.GetPagesBuilderDataResponse, error) {

	// Get site builder data
	data, err := s.siteBuilderDataRepository.GetBySiteID(ctx, siteID)
	if err != nil {
		log.Error(ctx, err)
		return nil, err
	}

	if data == nil {
		// Return empty data structure if no data exists
		response := &api.GetPagesBuilderDataResponse{
			SiteID:               siteID,
			SiteData:             map[string]interface{}{},
			VersionNumber:        0,
			LastPublishedVersion: nil,
			CreatedBy:            creatorID,
			CreatedAt:            time.Now().Unix(),
			UpdatedAt:            time.Now().Unix(),
		}
		return response, nil
	}

	// Parse JSON data
	var siteDataMap interface{}
	if len(data.SiteData) > 0 {
		if err := json.Unmarshal(data.SiteData, &siteDataMap); err != nil {
			log.Error(ctx, err)
			return nil, err
		}
	}

	response := &api.GetPagesBuilderDataResponse{
		SiteID:               data.SiteID,
		SiteData:             siteDataMap,
		VersionNumber:        data.VersionNumber,
		LastPublishedVersion: data.LastPublishedVersion,
		CreatedBy:            *data.CreatedBy,
		CreatedAt:            data.CreatedAt.Unix(),
		UpdatedAt:            data.UpdatedAt.Unix(),
	}

	log.AddNotice(ctx, "version_number", data.VersionNumber)
	log.AddNotice(ctx, "last_published_version", data.LastPublishedVersion)

	return response, nil
}

func (s *pagesBuilderService) PublishPagesBuilderData(ctx *gin.Context, creatorID string, req *api.PublishPagesBuilderDataRequest) (*api.PublishPagesBuilderDataResponse, error) {

	// Validate site ID
	if req.SiteID == "" {
		return nil, common.ErrSiteNotFound
	}

	// Check if site exists and belongs to creator
	exists, err := s.siteRepository.ExistsCreatorAndSiteID(ctx, creatorID, req.SiteID)
	if err != nil {
		log.Error(ctx, err)
		return nil, err
	}
	if !exists {
		log.Error(ctx, common.ErrSiteNotFound)
		return nil, common.ErrSiteNotFound
	}

	// Get current builder data to publish
	builderData, err := s.siteBuilderDataRepository.GetBySiteID(ctx, req.SiteID)
	if err != nil {
		log.Error(ctx, err)
		return nil, err
	}

	if builderData == nil {
		log.Error(ctx, common.ErrNoDataToPublish)
		return nil, common.ErrNoDataToPublish
	}

	// Check if this version is already published
	if builderData.LastPublishedVersion != nil && *builderData.LastPublishedVersion >= builderData.VersionNumber {
		// Get current published data to return existing info
		publishedData, err := s.siteCurrentPublishedRepository.GetBySiteID(ctx, req.SiteID)
		if err != nil {
			log.Error(ctx, err)
			return nil, err
		}
		if publishedData != nil {
			log.Warning(ctx, "No new data to publish, already at latest version")
			return &api.PublishPagesBuilderDataResponse{
				SiteID:        publishedData.SiteID,
				VersionNumber: publishedData.VersionNumber,
				PublishedAt:   publishedData.PublishedAt.Unix(),
			}, nil
		}
	}

	// Parse builder data to ensure it's valid JSON
	var siteDataMap interface{}
	if err := json.Unmarshal(builderData.SiteData, &siteDataMap); err != nil {
		log.Error(ctx, err)
		return nil, err
	}

	// Use transaction to publish data and update builder data
	var currentPublished *model.SiteCurrentPublished
	err = s.tx.Transaction(ctx, func(ctx context.Context) error {
		// Create history record first
		_, createHistoryErr := s.sitePublishedHistoryRepository.CreateHistoryRecord(ctx, req.SiteID, creatorID, builderData.VersionNumber, siteDataMap)
		if createHistoryErr != nil {
			return createHistoryErr
		}

		// Check if current published data exists
		existingCurrent, err := s.siteCurrentPublishedRepository.GetBySiteID(ctx, req.SiteID)
		if err != nil {
			return err
		}

		// Create or update current published data
		var createCurrentErr error
		if existingCurrent == nil {
			// Create new current published record
			currentPublished, createCurrentErr = s.siteCurrentPublishedRepository.CreateCurrentPublished(ctx, req.SiteID, creatorID, builderData.VersionNumber, siteDataMap)
		} else {
			// Update existing current published record
			currentPublished, createCurrentErr = s.siteCurrentPublishedRepository.UpdateCurrentPublished(ctx, req.SiteID, creatorID, builderData.VersionNumber, siteDataMap)
		}
		if createCurrentErr != nil {
			return createCurrentErr
		}

		// Update last published version in builder data
		builderData.LastPublishedVersion = &builderData.VersionNumber
		if err := s.siteBuilderDataRepository.Update(ctx, builderData); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		log.Error(ctx, err)
		return nil, err
	}

	response := &api.PublishPagesBuilderDataResponse{
		SiteID:        currentPublished.SiteID,
		VersionNumber: currentPublished.VersionNumber,
		PublishedAt:   currentPublished.PublishedAt.Unix(),
	}

	log.AddNotice(ctx, "version_number", currentPublished.VersionNumber)

	return response, nil
}

func (s *pagesBuilderService) GetPublishHistory(ctx *gin.Context, creatorID string, req *api.GetPublishHistoryRequest) (*api.GetPublishHistoryResponse, error) {

	// Validate site ID
	if req.SiteID == "" {
		log.Error(ctx, "Site ID is required")
		return nil, common.ErrSiteNotFound
	}

	// Check if site exists and belongs to creator
	exists, err := s.siteRepository.ExistsCreatorAndSiteID(ctx, creatorID, req.SiteID)
	if err != nil {
		log.Error(ctx, err)
		return nil, err
	}
	if !exists {
		log.Error(ctx, common.ErrSiteNotFound)
		return nil, common.ErrSiteNotFound
	}

	// Set default limit if not provided
	limit := req.Limit
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	offset := req.Offset
	if offset < 0 {
		offset = 0
	}

	// Get current published data to mark which version is current
	currentPublished, err := s.siteCurrentPublishedRepository.GetBySiteID(ctx, req.SiteID)
	if err != nil {
		log.Error(ctx, err)
		return nil, err
	}

	// Get publish history
	history, total, err := s.sitePublishedHistoryRepository.GetPublishHistory(ctx, req.SiteID, limit, offset)
	if err != nil {
		log.Error(ctx, err)
		return nil, err
	}

	// Convert to API response format
	var historyItems []api.PublishHistoryItem
	for _, item := range history {
		isCurrent := false
		if currentPublished != nil && item.VersionNumber == currentPublished.VersionNumber {
			isCurrent = true
		}

		publishedBy := ""
		if item.PublishedBy != nil {
			publishedBy = *item.PublishedBy
		}

		historyItems = append(historyItems, api.PublishHistoryItem{
			VersionNumber: item.VersionNumber,
			PublishedBy:   publishedBy,
			PublishedAt:   item.PublishedAt.Unix(),
			IsCurrent:     isCurrent,
		})
	}

	response := &api.GetPublishHistoryResponse{
		SiteID:  req.SiteID,
		History: historyItems,
		Total:   int(total),
	}

	log.AddNotice(ctx, "total", total)

	return response, nil
}

// GetSitePages gets the current published site pages data for public consumption
func (s *pagesBuilderService) GetSitePages(ctx *gin.Context, siteID string) (*api.GetSitePagesResponse, error) {

	// Validate site ID
	if siteID == "" {
		return nil, common.ErrSiteNotFound
	}

	// Get current published data
	publishedData, err := s.siteCurrentPublishedRepository.GetBySiteID(ctx, siteID)
	if err != nil {
		return nil, err
	}

	if publishedData == nil {
		return nil, common.ErrSiteNotFound
	}

	// Parse published data
	var publishedDataMap interface{}
	if len(publishedData.PublishedData) > 0 {
		if err := json.Unmarshal(publishedData.PublishedData, &publishedDataMap); err != nil {
			return nil, err
		}
	}

	response := &api.GetSitePagesResponse{
		SiteID:        publishedData.SiteID,
		VersionNumber: publishedData.VersionNumber,
		PublishedData: publishedDataMap,
		PublishedAt:   publishedData.PublishedAt.Unix(),
	}

	log.AddNotice(ctx, "version_number", publishedData.VersionNumber)
	log.AddNotice(ctx, "published_at", publishedData.PublishedAt)

	return response, nil
}

// ListTemplates returns available page templates with pagination
func (s *pagesBuilderService) ListTemplates(ctx *gin.Context, page, pageSize int) (*api.TemplateListResponseData, error) {
	items, total, err := s.sitePageTemplateRepository.List(ctx, page, pageSize)
	if err != nil {
		return nil, err
	}
	respItems := make([]api.TemplateItem, 0, len(items))
	for _, t := range items {
		respItems = append(respItems, api.TemplateItem{
			TemplateID:  t.TemplateID,
			Name:        t.Name,
			Description: t.Description,
			Cover:       t.Cover,
			Version:     1,
		})
	}
	return &api.TemplateListResponseData{
		Items:    respItems,
		Total:    int(total),
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (s *pagesBuilderService) TranslatePages(ctx *gin.Context, req *api.PageTranslateRequest) ([]*api.PageTranslateResponse, error) {
	var respItems []*api.PageTranslateResponse
	for _, items := range req.Items {
		var translations []api.PageTranslations

		res, err := s.translator.TranslatePage(translate.PageTranslateReq{
			PageI18nItem: translate.PageI18nItem{
				Name:        items.Texts.Name,
				Title:       items.Texts.Title,
				Description: items.Texts.Description,
				Keywords:    items.Texts.Keywords,
			},
		})
		if err != nil {
			log.Error(ctx, err)
			if strings.Contains(err.Error(), "handle translate error") {
				return nil, common.ErrHandleTranslateFailed
			}
			return nil, common.ErrValidateTranslateResult
		}
		for _, lang := range res {
			translations = append(translations, api.PageTranslations{
				Lang: lang.Language,
				PageTranslateTexts: api.PageTranslateTexts{
					Name:        lang.Name,
					Title:       lang.Title,
					Description: lang.Description,
					Keywords:    lang.Keywords,
				},
			})
		}

		respItems = append(respItems, &api.PageTranslateResponse{
			FieldType:    items.FieldType,
			Translations: translations,
			Context:      items.Context,
		})
	}
	return respItems, nil
}
