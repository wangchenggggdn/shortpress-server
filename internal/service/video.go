package service

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"shortpress-server/internal/api"
	"shortpress-server/internal/common"
	"shortpress-server/internal/model"
	"shortpress-server/internal/repository/db/creator"
	"shortpress-server/internal/repository/db/playlist"
	"shortpress-server/internal/repository/db/video"
	"shortpress-server/internal/types"
	"shortpress-server/pkg/log"
	"strconv"
	"strings"
	"time"

	"encoding/json"
	"os/exec"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const MAX_PLAYLIST_COUNT = 200

type VideoService interface {
	Upload(ctx *gin.Context, creatorID string, files []*multipart.FileHeader, playlistID string) ([]string, error)
	UploadSubtitle(ctx *gin.Context, creatorID string, files *multipart.FileHeader, vid string) (string, error)
	Replace(ctx *gin.Context, creatorID string, vid string, files *multipart.FileHeader) (string, string, error)
	// Add network video sources for a video
	AddSources(ctx *gin.Context, vid string, sources []*api.VideoSourceCreateItem) error
	// Batch fetch video information
	GetVideoByVIDs(ctx *gin.Context, vids []string) ([]*model.Video, error)
	GetVideoAndSeo(ctx *gin.Context, vid string) (*api.VideoInfo, error)
	// ListSources lists all playback sources for a video
	ListSources(ctx *gin.Context, vid string) ([]*api.VideoSourceInfo, error)
	ModifyVideo(ctx *gin.Context, req api.VideoInfo) error
	DeleteVideos(ctx *gin.Context, vids []string) error
	List(ctx *gin.Context, query *model.VideoQuery, page int, pageSize int, sortType int) ([]*model.Video, int64, error)
	UpdateUploadStatus(ctx *gin.Context, vid string, status int8) error
	RegenerateCover(ctx *gin.Context, creatorid string, playlistids string) ([]*types.ImageURL, error)
	GetVideoByVIDOnly(ctx *gin.Context, vid string) (*model.Video, error)
}

func NewVideoService(
	service *Service,
	videoRepository video.VideoRepository,
	videoSeoRepository video.VideoSeoRepository,
	videoSourceRepository video.VideoSourceRepository,
	playlistVidRepository playlist.PlaylistVidRepository,
	playlistService PlaylistService,
	creatorSiteRepository creator.CreatorSiteRepository,
	conf *viper.Viper,
) VideoService {
	return &videoService{
		Service:               service,
		videoRepository:       videoRepository,
		videoSeoRepository:    videoSeoRepository,
		videoSourceRepository: videoSourceRepository,
		playlistVidRepository: playlistVidRepository,
		playlistService:       playlistService,
		creatorSiteRepository: creatorSiteRepository,
		conf:                  conf,
	}
}

type videoService struct {
	*Service
	videoRepository       video.VideoRepository
	videoSeoRepository    video.VideoSeoRepository
	videoSourceRepository video.VideoSourceRepository
	playlistVidRepository playlist.PlaylistVidRepository
	playlistService       PlaylistService
	creatorSiteRepository creator.CreatorSiteRepository
	conf                  *viper.Viper
}

func (s *videoService) Upload(ctx *gin.Context, creatorID string, files []*multipart.FileHeader, playlistID string) ([]string, error) {

	// Create tasks first, then upload files
	vids := []string{}
	md5Hash := md5.Sum([]byte(creatorID))
	_ = md5Hash // no longer used for pathing
	uploadStatus := int(model.VideoUploadStatusUploading)
	queryParam := model.VideoQuery{
		CreatorID:    creatorID,
		UploadStatus: &uploadStatus,
	}

	count, err := s.videoRepository.Count(ctx, &queryParam)
	if err != nil {
		return nil, err
	}
	if count > 20 {
		log.Warning(ctx, fmt.Sprintf("creator %s upload videos count exceed limit, current count: %d", creatorID, count))
		return nil, common.ErrTooManyVideosUpload
	}

	for _, file := range files {
		vid := uuid.NewString()
		ext := filepath.Ext(file.Filename)
		_ = ext
		fileName := file.Filename[:len(file.Filename)-len(ext)]
		video := &model.VideoCore{
			VID:       vid,
			Title:     fileName,
			Status:    model.VideoStatusPublished,
			CreatorID: creatorID,
			Cover:     nil,
		}
		// TODO Consider the issue of creating 100 database entries if uploading 100 files. Batch processing should be considered later.
		err = s.tx.Transaction(ctx, func(ctx context.Context) error {
			err := s.videoRepository.Create(ctx, video)
			if err != nil {
				return err
			}
			err = s.videoSeoRepository.Create(ctx, &model.VideoSeo{
				VID:         vid,
				Title:       fileName,
				Description: fileName,
			})
			if err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			log.Error(ctx, fmt.Sprintf("create video failed: %s", err.Error()))
			return nil, err
		}
		// TODO Asynchronous file upload, batch file upload?

		// go func(f *multipart.FileHeader, videoPath string, vid string, playlistID string) {
		// 	newctx, cancel := context.WithTimeout(context.Background(), time.Minute*10)
		// 	defer cancel()
		// 	_, _, err := s.UploadOneFile(newctx, f, videoPath, vid, true, playlistID)
		// 	if err != nil {
		// 		s.logger.Warn("upload file failed", zap.Error(err))
		// 		err = s.UpdateUploadStatus(context.Background(), vid, model.VideoUploadStatusUploadFailed)
		// 		if err != nil {
		// 			s.logger.Warn("update upload status failed", zap.Error(err))
		// 		}
		// 	}
		// }(file, videoPath, vid, playlistID)

		// Synchronous upload for a single file into videolib/{creatorID}/{vid}/
		_, _, err := s.UploadOneFile(ctx, file, creatorID, vid, true, playlistID, "")
		if err != nil {
			log.Warning(ctx, fmt.Sprintf("upload file failed: %s", err.Error()))
		}

		vids = append(vids, vid)
		// 只使用一个
		break
	}
	return vids, nil
}

func (s *videoService) UploadSubtitle(ctx *gin.Context, creatorID string, file *multipart.FileHeader, vid string) (string, error) {
	// Check if the video exists
	video, err := s.videoRepository.GetByVID(ctx, vid)
	if err != nil {
		log.Error(ctx, fmt.Sprintf("get video by vid failed: %s", err.Error()))
		return "", err
	}
	if video == nil {
		log.Error(ctx, fmt.Sprintf("video not found: %s", vid))
		return "", common.ErrVideoNotFound
	}
	// Check if the file is a valid subtitle file
	if !strings.HasSuffix(file.Filename, ".srt") && !strings.HasSuffix(file.Filename, ".vtt") {
		log.Error(ctx, fmt.Sprintf("invalid subtitle file: %s", file.Filename))
		return "", fmt.Errorf("invalid subtitle file: %s", file.Filename)
	}
	// Create the destination file
	ext := filepath.Ext(file.Filename)
	md5Hash := md5.Sum([]byte(creatorID))
	creatorPath := hex.EncodeToString(md5Hash[:])
	subtitleUUID := uuid.NewString()
	subtitleFileName := vid + "/" + subtitleUUID + ext
	subtitlePath := "videolib/" + creatorPath + "/" + subtitleFileName
	subtitleFullPath := s.conf.GetString("storage.local.path") + "/" + subtitlePath

	// Ensure the directory exists
	if _, err := os.Stat(filepath.Dir(subtitleFullPath)); os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(subtitleFullPath), 0755); err != nil {
			log.Error(ctx, fmt.Sprintf("create directory failed: %s", err.Error()))
			return "", err
		}
	}

	// Create the destination file
	dst, err := os.Create(subtitleFullPath)
	if err != nil {
		log.Error(ctx, fmt.Sprintf("create subtitle file failed: %s", err.Error()))
		return "", err
	}
	defer dst.Close()

	// Open the source file
	src, err := file.Open()
	if err != nil {
		log.Error(ctx, fmt.Sprintf("open source file failed: %s", err.Error()))
		return "", err
	}
	defer src.Close()

	// Copy the file
	if _, err = io.Copy(dst, src); err != nil {
		log.Error(ctx, fmt.Sprintf("copy subtitle file failed: %s", err.Error()))
		return "", err
	}

	return subtitlePath, nil

}

type progressWriter struct {
	dst        io.Writer
	total      int64
	uploaded   int64
	lastUpdate time.Time
	vid        string
}

// Implement Write method to update progress
func (pw *progressWriter) Write(p []byte) (n int, err error) {
	n, err = pw.dst.Write(p)
	pw.uploaded += int64(n)

	// Update progress every 100ms or when progress changes more than 5% to avoid too frequent updates
	now := time.Now()
	if now.Sub(pw.lastUpdate) > 1000*time.Millisecond {
		pw.lastUpdate = now
		// progress := float64(pw.uploaded) * 100 / float64(pw.total)
		// fmt.Println("#########progress:", pw.vid, progress)
	}

	return n, err
}

// UploadOneFile uploads a single file for a video under path: videolib/{creatorID}/{vid}[/{subFolder}]/
// creatorID is used directly (no MD5). When subFolder is non-empty (e.g., "replace"),
// the file is stored under that subdirectory below the vid folder.
func (s *videoService) UploadOneFile(ctx *gin.Context, file *multipart.FileHeader, creatorID string, vid string, needCover bool, playlistID string, subFolder string) (string, string, error) {
	const maxRetries = 3
	retryCount := 0

	ext := filepath.Ext(file.Filename)
	vidFileName := vid + ext
	// Build relative destination directory: videolib/{creatorID}/{vid}[/{subFolder}]
	destRelDir := filepath.Join("videolib", creatorID, vid)
	if strings.TrimSpace(subFolder) != "" {
		destRelDir = filepath.Join(destRelDir, subFolder)
	}
	// Ensure destination directory exists before upload
	baseDir := filepath.Join(s.conf.GetString("storage.local.path"), destRelDir)
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return "", "", err
	}
	disFullName := filepath.Join(baseDir, vidFileName)
	dbCoverPath := filepath.Join(destRelDir, vid+".jpg")
	coverFullName := filepath.Join(baseDir, vid+".jpg")
	var err error

	// Create an initial video source record with status 'uploading' so we can update it precisely later
	sourceID := uuid.NewString()
	localPath := types.VideoUrl(filepath.Join(destRelDir, vidFileName))
	// pre-create source row
	preSrc := &model.VideoSource{
		SourceID:     sourceID,
		VID:          vid,
		Provider:     "local",
		SourceType:   1,
		URL:          nil,
		UploadStatus: model.VideoUploadStatusUploading,
		LocalPath:    &localPath,
		Duration:     0,
		Width:        0,
		Height:       0,
		Priority:     0,
		Status:       model.VideoSourceStatusEnabled,
	}
	if err := s.videoSourceRepository.Create(ctx, preSrc); err != nil {
		log.Error(ctx, fmt.Sprintf("create pre video source failed: %s", err.Error()))
	}

	for {

		// Create the destination file
		dst, err := os.Create(disFullName)
		if err != nil {
			s.logger.Warn("create file failed", zap.Error(err))
			break
		}
		defer func() {
			if dst != nil {
				if closeErr := dst.Close(); closeErr != nil {
					if err == nil { // If no other primary error occurred in this attempt
						err = closeErr // This close error becomes the function's error
					}
				}
			}
		}()

		// Open the source file
		src, err := file.Open()
		if err != nil {
			log.Error(ctx, fmt.Sprintf("open source file failed: %s", err.Error()))
			break
		}
		defer func() {
			if src != nil {
				if closeErr := src.Close(); closeErr != nil {
					log.Error(ctx, fmt.Sprintf("failed to close source file: %s", closeErr.Error()))
					if err == nil { // If no other primary error occurred in this attempt
						err = closeErr
					}
				}
			}
		}()

		pw := &progressWriter{
			dst:        dst,
			total:      file.Size,
			uploaded:   0,
			lastUpdate: time.Now(),
			vid:        vid,
		}

		if _, err = io.Copy(pw, src); err != nil {
			retryCount++
			if retryCount >= maxRetries {
				log.Warning(ctx, fmt.Sprintf("retry count exceed limit: %d", retryCount))
				break
			}
			// Wait for a short period before retrying
			time.Sleep(time.Second * time.Duration(retryCount))
			continue
		}

		// Generate cover (video upload still succeeds if cover extraction fails; do not persist a URL without a file)
		if needCover {
			if err = s.saveCoverFromVideo(ctx, disFullName, coverFullName); err != nil {
				log.Warning(ctx, fmt.Sprintf("save cover failed, video saved without cover: %s", err.Error()))
				dbCoverPath = ""
			} else if _, statErr := os.Stat(coverFullName); statErr != nil {
				log.Warning(ctx, fmt.Sprintf("cover file missing after extract: %s", statErr.Error()))
				dbCoverPath = ""
			}
		} else {
			dbCoverPath = ""
		}
		err = nil
		break
	}
	// Update video upload result
	if err != nil {
		log.Error(ctx, fmt.Sprintf("upload file failed: %s, vid: %s", err.Error(), vid))
		err1 := s.videoSourceRepository.UpdateUploadStatusBySourceID(ctx, sourceID, model.VideoUploadStatusUploadFailed)
		if err1 != nil {
			log.Error(ctx, fmt.Sprintf("update upload status failed: %s", err1.Error()))
		}
		return "", "", err
	} else {
		// Update the pre-created local video source with success and metadata
		vp := types.VideoUrl(filepath.Join(destRelDir, vidFileName))
		vc := types.ImageURL(dbCoverPath)
		duration, width, height := s.getVideoMetadata(ctx, disFullName)
		// Save cover only when a file exists on disk (path non-empty)
		if needCover && dbCoverPath != "" {
			if err := s.videoRepository.Update(ctx, &model.Video{VID: vid, Cover: &vc}); err != nil {
				log.Error(ctx, fmt.Sprintf("update video cover failed: %s", err.Error()))
			}
		}
		// Update the existing source row
		updates := map[string]interface{}{
			"upload_status": model.VideoUploadStatusUploadSuccess,
			"local_path":    vp,
			"duration":      duration,
			"width":         width,
			"height":        height,
		}
		if err := s.videoSourceRepository.UpdateBySourceID(ctx, sourceID, updates); err != nil {
			log.Error(ctx, fmt.Sprintf("update video source failed: %s", err.Error()))
		}
		if playlistID != "" {
			// Cancel the use of playlist_video table
			err := s.playlistVidRepository.Create(ctx, &model.PlaylistVid{
				PlaylistID: playlistID,
				VID:        vid,
			})
			if err != nil {
				log.Error(ctx, fmt.Sprintf("add video to playlist failed, but video upload success: %s", err.Error()))
			}

			orderData, err := s.playlistService.GetVideosOrder(ctx, playlistID)
			if err == nil || orderData != nil {
				if len(orderData.SortData.VIDs) > MAX_PLAYLIST_COUNT {
					log.Warning(ctx, fmt.Sprintf("videos count exceed limit: %d, playlistID: %s", len(orderData.SortData.VIDs), playlistID))
				} else {
					err = s.playlistService.AppendVidsToOrder(ctx, playlistID, []string{vid})
					if err != nil {
						log.Warning(ctx, fmt.Sprintf("append video to order videos failed, but video upload success: %s", err.Error()))
					}
				}
			}

		}
		return string(vp), dbCoverPath, nil
	}

}

func (s *videoService) List(ctx *gin.Context, query *model.VideoQuery, page int, pageSize int, sortType int) ([]*model.Video, int64, error) {
	if query.ExcludePlaylistID != "" {
		// s.logger.Warn("exclude playlist id", zap.String("playlistID", query.ExcludePlaylistID))
		orderData, err := s.playlistService.GetVideosOrder(ctx, query.ExcludePlaylistID)
		if err == nil || orderData != nil {
			query.ExcludeVids = orderData.SortData.VIDs
		}
	}
	if query.SiteID != "" {
		creatorID, err := s.creatorSiteRepository.FindCreatorBySitID(ctx, query.SiteID)
		if err != nil || creatorID == "" {
			return nil, 0, err
		}
		query.CreatorID = creatorID
	}

	return s.videoRepository.ListByPageV2(ctx, query, page, pageSize, sortType)

}

func (s *videoService) GetVideoAndSeo(ctx *gin.Context, vid string) (*api.VideoInfo, error) {
	video, seo, err := s.videoRepository.GetVideoAndSeo(ctx, vid)
	if err != nil {
		log.Error(ctx, fmt.Sprintf("get video and seo failed: %s", err.Error()))
		return nil, err
	}
	if video == nil {
		log.Error(ctx, fmt.Sprintf("video not found: %s", vid))
		return nil, common.ErrVideoNotFound
	}

	vidoInfo := &api.VideoInfo{
		VID:         video.VID,
		Title:       video.Title,
		Description: video.Description,
		Tags:        video.Tags,
		Cover:       video.Cover,
		Status:      video.Status,
		CreatedAt:   video.CreatedAt.Unix(),
		UpdatedAt:   video.UpdatedAt.Unix(),
		Subtitles:   video.Subtitles,
		Config:      video.Config,
	}
	// Populate all sources for this video
	if sources, err := s.videoSourceRepository.ListByVID(ctx, vid); err == nil {
		list := make([]*api.VideoSourceInfo, 0, len(sources))
		for _, src := range sources {
			// Build URL: prefer explicit external URL; otherwise expand local path with configured host
			urlStr := ""
			if src.URL != nil && *src.URL != "" {
				urlStr = *src.URL
			} else if src.LocalPath != nil {
				base := strings.TrimRight(s.conf.GetString("storage.local.video_host"), "/")
				p := string(*src.LocalPath)
				if p != "" && !strings.HasPrefix(p, "/") {
					p = "/" + p
				}
				urlStr = base + p
			}
			list = append(list, &api.VideoSourceInfo{
				SourceID:     src.SourceID,
				Provider:     src.Provider,
				SourceType:   int(src.SourceType),
				URL:          urlStr,
				UploadStatus: src.UploadStatus,
				Priority:     src.Priority,
				Duration:     src.Duration,
				Width:        src.Width,
				Height:       src.Height,
			})
		}
		vidoInfo.Sources = list
	} else {
		log.Warning(ctx, fmt.Sprintf("list sources failed for vid %s: %v", vid, err))
	}
	if seo != nil {
		vidoInfo.Seo = &api.VideoSeo{
			Title:       seo.Title,
			Description: seo.Description,
			Keywords:    seo.Keywords,
		}
	}
	log.AddNotice(ctx, "video_title", video.Title)
	return vidoInfo, nil
}

// Modify video information
func (s *videoService) ModifyVideo(ctx *gin.Context, req api.VideoInfo) error {
	video := &model.Video{
		VID:         req.VID,
		Title:       req.Title,
		Description: req.Description,
		Tags:        req.Tags,
		Cover:       req.Cover,
		Status:      req.Status,
		Subtitles:   req.Subtitles, //TODO
		Config:      req.Config,
	}
	err := s.videoRepository.Update(ctx, video)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return common.ErrVideoNotFound
		}
		return err
	}
	if req.Seo != nil {
		err := s.videoSeoRepository.Save(ctx, &model.VideoSeo{
			VID:         req.VID,
			Title:       req.Seo.Title,
			Description: req.Seo.Description,
			Keywords:    req.Seo.Keywords,
		})
		if err != nil {
			return err
		}
	}
	// Reconcile playback sources if provided
	if req.Sources != nil {
		// Fetch current active sources
		existing, err := s.videoSourceRepository.ListByVID(ctx, req.VID)
		if err != nil {
			return err
		}
		existMap := make(map[string]*model.VideoSource, len(existing))
		for _, e := range existing {
			existMap[e.SourceID] = e
		}

		keepIDs := make([]string, 0, len(req.Sources))
		// Process updates/creates
		for _, in := range req.Sources {
			if in == nil {
				continue
			}
			if strings.TrimSpace(in.SourceID) != "" {
				// Update existing by source_id
				keepIDs = append(keepIDs, in.SourceID)
				fields := map[string]interface{}{
					"provider":    in.Provider,
					"source_type": in.SourceType,
					"priority":    in.Priority,
					"duration":    in.Duration,
					"width":       in.Width,
					"height":      in.Height,
				}
				// URL vs LocalPath
				u := strings.TrimSpace(in.URL)
				if u != "" {
					if in.SourceType == 1 || strings.ToLower(in.Provider) == "local" {
						lp := types.VideoUrl(u)
						fields["local_path"] = lp
						fields["url"] = gorm.Expr("NULL")
					} else {
						fields["url"] = u
						fields["local_path"] = gorm.Expr("NULL")
					}
				}
				// UploadStatus (optional)
				if in.UploadStatus != 0 {
					fields["upload_status"] = in.UploadStatus
				}
				if err := s.videoSourceRepository.UpdateBySourceID(ctx, in.SourceID, fields); err != nil {
					return err
				}
			} else {
				// Create new source
				sid := uuid.NewString()
				newEntity := &model.VideoSource{
					SourceID:     sid,
					VID:          req.VID,
					Provider:     in.Provider,
					SourceType:   int8(in.SourceType),
					UploadStatus: in.UploadStatus,
					Priority:     in.Priority,
					Duration:     in.Duration,
					Width:        in.Width,
					Height:       in.Height,
					Status:       model.VideoSourceStatusEnabled,
				}
				u := strings.TrimSpace(in.URL)
				if u != "" {
					if in.SourceType == 1 || strings.ToLower(in.Provider) == "local" {
						lp := types.VideoUrl(u)
						newEntity.LocalPath = &lp
					} else {
						newEntity.URL = &u
					}
				}
				if newEntity.UploadStatus == 0 {
					// assume success for external, or keep 0 if unspecified
					if newEntity.URL != nil {
						newEntity.UploadStatus = model.VideoUploadStatusUploadSuccess
					}
				}
				if err := s.videoSourceRepository.Create(ctx, newEntity); err != nil {
					return err
				}
				keepIDs = append(keepIDs, sid)
			}
		}
		// Delete any existing sources not present in the request (treat request as desired state)
		if err := s.videoSourceRepository.DelNotIn(ctx, req.VID, keepIDs); err != nil {
			return err
		}
	}
	log.AddNotice(ctx, "video_title", req.Title)
	return nil
}

// Delete videos
func (s *videoService) DeleteVideos(ctx *gin.Context, vids []string) error {
	err := s.tx.Transaction(ctx, func(ctx context.Context) error {
		if err := s.videoRepository.BatchDelete(ctx, vids); err != nil {
			return err
		}
		// No longer delete associated IDs
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

func (s *videoService) UpdateUploadStatus(ctx *gin.Context, vid string, status int8) error {
	return s.videoRepository.UpdateUploadStatus(ctx, vid, status)
}

// Use FFprobe + FFmpeg to get keyframes from a video file
func (s *videoService) saveCoverFromVideo(ctx *gin.Context, videoFile string, savePath string) error {

	if _, err := os.Stat(filepath.Dir(savePath)); os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(savePath), 0755); err != nil {
			return err
		}
	}

	// Extract one frame as JPEG (avoid invalid flags that cause silent failure on some ffmpeg builds)
	extractCmd := exec.CommandContext(ctx, "ffmpeg",
		"-hide_banner", "-loglevel", "error",
		"-i", videoFile,
		"-vframes", "1",
		"-q:v", "5",
		"-y",
		savePath,
	)
	if output, err := extractCmd.CombinedOutput(); err != nil {
		log.Error(ctx, fmt.Sprintf("extract cover failed: %s, output: %s", err.Error(), string(output)))
		return err
	}
	return nil
}

func (s *videoService) getVideoMetadata(ctx *gin.Context, videoFile string) (int64, int32, int32) {
	cmd := exec.CommandContext(ctx, "ffprobe",
		"-v", "quiet",
		"-select_streams", "v:0",
		"-show_entries", "stream=width,height:format=duration",
		"-of", "json",
		videoFile,
	)
	output, err := cmd.Output()
	if err != nil {
		log.Error(ctx, fmt.Sprintf("ffprobe run failed: %s", err.Error()))
		return 0, 0, 0
	}

	var result struct {
		Streams []struct {
			Width  int32 `json:"width"`
			Height int32 `json:"height"`
		} `json:"streams"`
		Format struct {
			Duration string `json:"duration"`
		} `json:"format"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		log.Error(ctx, fmt.Sprintf("parse ffprobe output failed: %s", err.Error()))
		return 0, 0, 0
	}

	var duration int64
	if result.Format.Duration != "" {
		if d, err := strconv.ParseFloat(result.Format.Duration, 64); err == nil {
			duration = int64(d)
		}
	}

	var width, height int32
	if len(result.Streams) > 0 {
		width = result.Streams[0].Width
		height = result.Streams[0].Height
	}

	return duration, width, height
}

// Replace data only, vid remains unchanged
func (s *videoService) Replace(ctx *gin.Context, creatorID string, vid string, file *multipart.FileHeader) (string, string, error) {
	video, err := s.videoRepository.GetByVID(ctx, vid)
	if err != nil {
		return "", "", err
	}
	if video == nil {
		log.Error(ctx, fmt.Sprintf("video not found: %s", vid))
		return "", "", common.ErrVideoNotFound
	}
	// Store replacement under videolib/{creatorID}/{vid}/replace/
	return s.UploadOneFile(ctx, file, creatorID, vid, false, "", "replace")
}

func (s *videoService) GetVideoByVIDs(ctx *gin.Context, vids []string) ([]*model.Video, error) {
	result, err := s.videoRepository.GetByVIDs(ctx, vids)
	if err != nil {
		log.Error(ctx, fmt.Sprintf("get videos by VIDs failed: %s", err.Error()))
		return nil, err
	}

	// Create a map for quick lookup
	videoMap := make(map[string]*model.Video)
	for _, video := range result {
		videoMap[video.VID] = video
	}

	// Build ordered result according to input vids order
	orderedResult := make([]*model.Video, 0, len(vids))
	for _, vid := range vids {
		if video, exists := videoMap[vid]; exists {
			orderedResult = append(orderedResult, video)
		}
	}

	return orderedResult, nil
}

// AddSources adds multiple network sources for a video
func (s *videoService) AddSources(ctx *gin.Context, vid string, sources []*api.VideoSourceCreateItem) error {
	// ensure video exists
	v, err := s.videoRepository.GetByVID(ctx, vid)
	if err != nil {
		return err
	}
	if v == nil {
		return common.ErrVideoNotFound
	}
	// build entities
	items := make([]*model.VideoSource, 0, len(sources))
	for i, src := range sources {
		if src.Provider == "" {
			src.Provider = "external"
		}
		// default priority start from 10 to leave room for local=1
		prio := src.Priority
		if prio == 0 {
			prio = 10 + i
		}
		url := src.URL
		entity := &model.VideoSource{
			SourceID:     uuid.NewString(),
			VID:          vid,
			Provider:     src.Provider,
			SourceType:   int8(src.SourceType),
			URL:          &url,
			UploadStatus: model.VideoUploadStatusUploadSuccess,
			LocalPath:    nil,
			Duration:     0,
			Width:        0,
			Height:       0,
			// Subtitles:  src.Subtitles,
			Priority: prio,
			Status:   model.VideoSourceStatusEnabled,
		}
		items = append(items, entity)
	}
	return s.videoSourceRepository.BatchCreate(ctx, items)
}

// ListSources lists all sources for a given video
func (s *videoService) ListSources(ctx *gin.Context, vid string) ([]*api.VideoSourceInfo, error) {
	list, err := s.videoSourceRepository.ListByVID(ctx, vid)
	if err != nil {
		return nil, err
	}
	result := make([]*api.VideoSourceInfo, 0, len(list))
	for _, src := range list {
		// Build URL string (external preserved; local expanded with host)
		urlStr := ""
		if src.URL != nil && *src.URL != "" {
			urlStr = *src.URL
		} else if src.LocalPath != nil {
			base := strings.TrimRight(s.conf.GetString("storage.local.video_host"), "/")
			p := string(*src.LocalPath)
			if p != "" && !strings.HasPrefix(p, "/") {
				p = "/" + p
			}
			urlStr = base + p
		}
		result = append(result, &api.VideoSourceInfo{
			SourceID:     src.SourceID,
			Provider:     src.Provider,
			SourceType:   int(src.SourceType),
			URL:          urlStr,
			UploadStatus: src.UploadStatus,
			Priority:     src.Priority,
			Duration:     src.Duration,
			Width:        src.Width,
			Height:       src.Height,
		})
	}
	return result, nil
}

func (s *videoService) RegenerateCover(ctx *gin.Context, creatorid string, playlistids string) ([]*types.ImageURL, error) {

	// // playlistidsList := []string{}
	// if playlistids == "" {
	// 	log.Warning(ctx, "playlistids is empty, return")
	// 	return nil, fmt.Errorf("playlistids is empty")

	// }
	// playlistidsList := strings.Split(playlistids, ",")
	// covers := make([]*types.ImageURL, 0, len(playlistidsList))
	// for _, playlistid := range playlistidsList {
	// 	if playlistid == "" {
	// 		continue
	// 	}
	// 	orderData, err := s.playlistService.GetVideosOrder(ctx, playlistid)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// 	if orderData == nil || len(orderData.SortData.VIDs) == 0 {
	// 		continue
	// 	}
	// 	for _, vid := range orderData.SortData.VIDs {

	// 		video, err := s.videoRepository.GetByVID(ctx, vid)
	// 		if err != nil {
	// 			continue
	// 		}
	// 		if video.Cover == nil  {
	// 			continue
	// 		}

	// 		videoFullName := s.conf.GetString("storage.local.path") + "/" + string(*video.VideoPath)
	// 		coverFullName := s.conf.GetString("storage.local.path") + "/" + string(*video.Cover)
	// 		if err = s.saveCoverFromVideo(ctx, videoFullName, coverFullName); err != nil {
	// 			log.Error(ctx, fmt.Sprintf("save cover failed for video %s: %s", vid, err.Error()))
	// 			break
	// 		}
	// 		covers = append(covers, video.Cover)
	// 	}

	// }
	// return covers, nil
	return nil, nil
}

// GetVideoByVIDOnly gets video config by VID (internal API use, no SEO join)
func (s *videoService) GetVideoByVIDOnly(ctx *gin.Context, vid string) (*model.Video, error) {
	return s.videoRepository.GetByVID(ctx, vid)
}
