package service

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"time"

	"shortpress-server/internal/api"
	"shortpress-server/internal/common"
	"shortpress-server/internal/model"
	"shortpress-server/internal/repository/db/creator"
	"shortpress-server/internal/repository/db/playlist"
	"shortpress-server/internal/repository/db/site"
	"shortpress-server/internal/repository/db/video"
	"shortpress-server/internal/types"
	"shortpress-server/pkg/log"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/spf13/viper"
	"golang.org/x/crypto/bcrypt"
)

var CreatorGuides = []string{"ADD_FIRST_PLAYLIST", "UPLOAD_VIDEO", "ADD_VIDEO_TO_PLAYLIST"}

type CreatorService interface {
	GetCreator(ctx context.Context, id string) (*model.Creator, error)
	RegisterCreator(ctx context.Context, req *api.RegisterCreatorRequest) error
	LoginCreator(ctx context.Context, req *api.CreatorLoginRequest) (string, error)
	GetCreatorProfile(ctx context.Context, creatorID string) (*model.CreatorProfile, error)
	GetGuides(ctx context.Context, creatorID string) []*api.CreatorGuides
	UploadImg(ctx *gin.Context, creatorID string, file *multipart.FileHeader) (*types.ImageURL, error)
	GetStatsCount(ctx context.Context, creatorID string) (int64, int64, int64)
	CompleteGuides(ctx *gin.Context, creatorID string, guides []string) error
}

func NewCreatorService(
	service *Service,
	creatorRepository creator.CreatorRepository,
	creatorProfileRepository creator.CreatorProfileRepository,
	creatorSiteRepository creator.CreatorSiteRepository,
	creatorGuidesRepository creator.CreatorGuidesRepository,
	siteRepository site.SiteRepository,
	playlistRepository playlist.PlaylistRepository,
	videoRepository video.VideoRepository,
	conf *viper.Viper,
) CreatorService {
	return &creatorService{
		Service:                  service,
		creatorRepository:        creatorRepository,
		creatorProfileRepository: creatorProfileRepository,
		creatorSiteRepository:    creatorSiteRepository,
		creatorGuidesRepository:  creatorGuidesRepository,
		siteRepository:           siteRepository,
		playlistRepository:       playlistRepository,
		videoRepository:          videoRepository,
		conf:                     conf,
	}
}

type creatorService struct {
	*Service
	creatorRepository        creator.CreatorRepository
	creatorProfileRepository creator.CreatorProfileRepository
	creatorSiteRepository    creator.CreatorSiteRepository
	creatorGuidesRepository  creator.CreatorGuidesRepository
	siteRepository           site.SiteRepository
	conf                     *viper.Viper
	playlistRepository       playlist.PlaylistRepository
	videoRepository          video.VideoRepository
}

func (s *creatorService) GetCreator(ctx context.Context, id string) (*model.Creator, error) {
	return s.creatorRepository.GetByCreatorID(ctx, id)
}

func (s *creatorService) RegisterCreator(ctx context.Context, req *api.RegisterCreatorRequest) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	if r, err := s.creatorRepository.GetByEmail(ctx, req.Email); err != nil {
		return err
	} else if r != nil {
		return common.ErrEmailOrNameAlreadyUse
	}

	creator := &model.Creator{
		CreatorID:    uuid.NewString(),
		Email:        req.Email,
		PasswordHash: string(hash),
		Type:         0, // Default type
		Role:         1, // Default role, 1: other
		Status:       0, // Default not activated
	}
	profiles := &model.CreatorProfile{
		CreatorID: creator.CreatorID,
		Nickname:  "",
		AvatarURL: "", //TODO Default avatar
	}

	err = s.tx.Transaction(ctx, func(ctx context.Context) error {
		if err = s.creatorRepository.Create(ctx, creator); err != nil {
			return err
		}
		if err = s.creatorProfileRepository.Create(ctx, profiles); err != nil {
			return err
		}
		return nil
	})

	return err
}

func (s *creatorService) LoginCreator(ctx context.Context, req *api.CreatorLoginRequest) (string, error) {
	creator, err := s.creatorRepository.GetByEmail(ctx, req.Email)
	if err != nil {
		return "", err
	}
	if creator == nil {
		return "", common.ErrCreatorNotFound
	}
	if err := bcrypt.CompareHashAndPassword([]byte(creator.PasswordHash), []byte(req.Password)); err != nil {
		return "", common.ErrInvalidCredential
	}
	// 7 days validity period
	token, err := s.jwt.GenToken(creator.CreatorID, time.Now().Add(time.Hour*24*7))
	if err != nil {
		return "", err
	}
	return token, nil
}

func (s *creatorService) GetCreatorProfile(ctx context.Context, creatorID string) (*model.CreatorProfile, error) {
	return s.creatorProfileRepository.GetProfileByCreatorID(ctx, creatorID)
}

func (s *creatorService) UploadImg(ctx *gin.Context, creatorID string, file *multipart.FileHeader) (*types.ImageURL, error) {

	// Storage path is base_path/md5sum(creator_id)/res/img/xxx.jpg
	md5Hash := md5.Sum([]byte(creatorID))
	filenName := uuid.NewString() + filepath.Ext(file.Filename)
	imgPath := "res/img/" + hex.EncodeToString(md5Hash[:]) + "/"
	fullPath := s.conf.GetString("storage.local.path") + "/" + imgPath
	imgFullName := fullPath + "/" + filenName
	// Check if directory exists, if not create it
	if _, err := os.Stat(filepath.Dir(imgFullName)); os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(imgFullName), 0755); err != nil {
			return nil, err
		}
	}
	// Create directory
	dst, err := os.Create(imgFullName)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := dst.Close(); closeErr != nil {
			if err == nil { // Only overwrite err if no previous error occurred
				err = closeErr
			}
		}
	}()

	src, err := file.Open()
	if err != nil {
		log.Error(ctx, "file open error: "+err.Error())
		return nil, err
	}
	defer func() {
		if closeErr := src.Close(); closeErr != nil {
			if err == nil { // Only overwrite err if no previous error occurred
				err = closeErr
			}
		}
	}()

	if _, err = io.Copy(dst, src); err != nil {
		return nil, err
	}
	// return s.conf.GetString("storage.local.image_host") + "/" +  imgPath + filenName, nil
	imgUrl := types.ImageURL(imgPath + filenName)
	return &imgUrl, nil

}

func (s *creatorService) GetStatsCount(ctx context.Context, creatorID string) (int64, int64, int64) {
	// Count sites TODO pending query
	sitesCount, err := s.creatorSiteRepository.Count(ctx, creatorID)
	if err != nil {
		sitesCount = 0
	}
	// Count playlists
	playlistsCount, err := s.playlistRepository.Count(ctx, creatorID)
	if err != nil {
		playlistsCount = 0
	}

	// Count videos
	ps := int(-1)
	queryParam := &model.VideoQuery{
		CreatorID: creatorID,
		Status:    &ps,
	}
	videosCount, err := s.videoRepository.Count(ctx, queryParam)
	if err != nil {
		videosCount = 0
	}

	return sitesCount, playlistsCount, videosCount
}

func (s *creatorService) GetGuides(ctx context.Context, creatorID string) []*api.CreatorGuides {
	var result []*api.CreatorGuides

	guides, err := s.creatorGuidesRepository.GetByCreatorID(ctx, creatorID)
	if err != nil {
		return result
	}

	flag := false
	// TODO Pending optimization
	for _, g := range CreatorGuides {
		for _, dbg := range guides {
			if dbg.Guides == g {
				flag = true
				break
			}
		}
		if !flag {
			result = append(result, &api.CreatorGuides{
				Name:   g,
				Status: 0,
			})
		} else {
			result = append(result, &api.CreatorGuides{
				Name:   g,
				Status: 1,
			})
			flag = false
		}
	}
	return result
}

// CompleteGuides
func (s *creatorService) CompleteGuides(ctx *gin.Context, creatorID string, guides []string) error {
	// TODO Pending optimization
	guidMap := make(map[string]int)
	for _, g := range CreatorGuides {
		guidMap[g] = 1
	}
	var appendGuides []*model.CreatorGuides
	for _, g := range guides {
		if _, exists := guidMap[g]; !exists {
			return common.ErrInvalidGuides
		}
		appendGuides = append(appendGuides, &model.CreatorGuides{
			CreatorID: creatorID,
			Guides:    g,
		})
	}
	err := s.creatorGuidesRepository.Create(ctx, appendGuides)
	if err != nil {
		log.Warning(ctx, "create guides error: "+err.Error())
	}
	return nil
}
