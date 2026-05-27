package analytics

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"shortpress-server/internal/api"
	"shortpress-server/internal/model"
	"shortpress-server/internal/repository/db/user"
)

// MergeMetaIntoSnapshot stores meta click ids on the payment transaction snapshot.
func MergeMetaIntoSnapshot(snapshot model.JSONMap, meta *api.MetaClickPayload) model.JSONMap {
	if snapshot == nil {
		snapshot = model.JSONMap{}
	}
	if meta == nil {
		return snapshot
	}
	if v := strings.TrimSpace(meta.Fbc); v != "" {
		snapshot["meta_fbc"] = v
	}
	if v := strings.TrimSpace(meta.Fbp); v != "" {
		snapshot["meta_fbp"] = v
	}
	if v := strings.TrimSpace(meta.Fbclid); v != "" {
		snapshot["meta_fbclid"] = v
	}
	if v := strings.TrimSpace(meta.EventSourceURL); v != "" {
		snapshot["meta_event_source_url"] = v
	}
	snapshot["meta_captured_at"] = time.Now().Unix()
	return snapshot
}

// PersistUserMetaClick writes the latest fbc/fbp onto the user row (DB-backed attribution).
func PersistUserMetaClick(ctx context.Context, userRepo user.UserRepository, userID string, meta *api.MetaClickPayload) error {
	if userRepo == nil || userID == "" || meta == nil {
		return nil
	}
	fbc := strings.TrimSpace(meta.Fbc)
	fbp := strings.TrimSpace(meta.Fbp)
	fbclid := strings.TrimSpace(meta.Fbclid)
	if fbc == "" && fbp == "" && fbclid == "" {
		return nil
	}
	if fbc == "" && fbclid != "" {
		fbc = BuildFbcFromFbclid(fbclid)
	}
	return userRepo.UpdateMetaClickIds(ctx, userID, fbc, fbp, fbclid)
}

// BuildFbcFromFbclid formats Meta _fbc from a raw fbclid query value.
func BuildFbcFromFbclid(fbclid string) string {
	fbclid = strings.TrimSpace(fbclid)
	if fbclid == "" {
		return ""
	}
	return fmt.Sprintf("fb.1.%d.%s", time.Now().UnixMilli(), fbclid)
}

// ResolvedMetaClick holds fbc/fbp resolved for CAPI from snapshot + user.
type ResolvedMetaClick struct {
	Fbc            string
	Fbp            string
	EventSourceURL string
}

// ApplyMetaAttributionProperties merges Meta click ids into analytics event properties (e.g. 数数 purchase).
func ApplyMetaAttributionProperties(props map[string]interface{}, meta ResolvedMetaClick, snapshot model.JSONMap) {
	if props == nil {
		return
	}
	if meta.Fbc != "" {
		props["fbc"] = meta.Fbc
	}
	if meta.Fbp != "" {
		props["fbp"] = meta.Fbp
	}
	if meta.EventSourceURL != "" {
		props["event_source_url"] = meta.EventSourceURL
	}
	if snapshot != nil {
		if fbclid := stringFromSnapshot(snapshot, "meta_fbclid"); fbclid != "" {
			props["fbclid"] = fbclid
		}
	}
}

// ResolveMetaClick prefers transaction snapshot, then user DB fields.
func ResolveMetaClick(snapshot model.JSONMap, u *model.User) ResolvedMetaClick {
	var out ResolvedMetaClick
	if snapshot != nil {
		out.Fbc = stringFromSnapshot(snapshot, "meta_fbc")
		out.Fbp = stringFromSnapshot(snapshot, "meta_fbp")
		out.EventSourceURL = stringFromSnapshot(snapshot, "meta_event_source_url")
		if out.Fbc == "" {
			if fbclid := stringFromSnapshot(snapshot, "meta_fbclid"); fbclid != "" {
				out.Fbc = BuildFbcFromFbclid(fbclid)
			}
		}
	}
	if u != nil {
		if out.Fbc == "" {
			out.Fbc = strings.TrimSpace(u.MetaFbc)
		}
		if out.Fbp == "" {
			out.Fbp = strings.TrimSpace(u.MetaFbp)
		}
		if out.Fbc == "" && strings.TrimSpace(u.MetaFbclid) != "" {
			out.Fbc = BuildFbcFromFbclid(u.MetaFbclid)
		}
	}
	return out
}

func stringFromSnapshot(snapshot model.JSONMap, key string) string {
	if snapshot == nil {
		return ""
	}
	v, ok := snapshot[key]
	if !ok || v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	default:
		return strings.TrimSpace(fmt.Sprint(t))
	}
}

// HashMetaPII SHA-256 hashes normalized PII per Meta CAPI requirements.
func HashMetaPII(value string) string {
	normalized := strings.TrimSpace(strings.ToLower(value))
	if normalized == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(sum[:])
}
