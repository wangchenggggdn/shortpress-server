package analytics

import (
	"database/sql"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"shortpress-server/internal/model"
)

// TestLocalMetaCapiFlow 在 scripts/test_meta_capi_flow.sh 设置 ORDER_ID/USER_ID 后验证 ResolveMetaClick。
func TestLocalMetaCapiFlow(t *testing.T) {
	orderID := os.Getenv("ORDER_ID")
	userID := os.Getenv("USER_ID")
	if orderID == "" || userID == "" {
		t.Skip("set ORDER_ID and USER_ID (run scripts/test_meta_capi_flow.sh first)")
	}

	dsn := os.Getenv("TEST_MYSQL_DSN")
	if dsn == "" {
		dsn = "root:shortpress@tcp(127.0.0.1:3306)/shortpress?charset=utf8mb4&parseTime=true&loc=Local"
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	var snapshotJSON []byte
	var siteID, currency string
	var amount int64
	var status int
	var createdAt time.Time
	err = db.QueryRow(`
		SELECT site_id, amount, currency, status, snapshot, created_at
		FROM payment_transactions WHERE transaction_id = ?`, orderID).
		Scan(&siteID, &amount, &currency, &status, &snapshotJSON, &createdAt)
	if err != nil {
		t.Fatalf("load transaction: %v", err)
	}
	if status != model.PaymentStatusSuccess {
		t.Fatalf("expected paid status=2, got %d", status)
	}

	var snapshot model.JSONMap
	if err := json.Unmarshal(snapshotJSON, &snapshot); err != nil {
		t.Fatalf("snapshot json: %v", err)
	}

	var u model.User
	err = db.QueryRow(`
		SELECT user_id, email, meta_fbc, meta_fbp, meta_fbclid
		FROM users WHERE user_id = ?`, userID).
		Scan(&u.UserID, &u.Email, &u.MetaFbc, &u.MetaFbp, &u.MetaFbclid)
	if err != nil {
		t.Fatalf("load user: %v", err)
	}

	meta := ResolveMetaClick(snapshot, &u)
	if meta.Fbc == "" || meta.Fbp == "" {
		t.Fatalf("ResolveMetaClick empty: fbc=%q fbp=%q", meta.Fbc, meta.Fbp)
	}
	if want := os.Getenv("TEST_FBC"); want != "" && meta.Fbc != want {
		t.Errorf("fbc mismatch: got %q want %q", meta.Fbc, want)
	}
	t.Logf("ResolveMetaClick OK: fbc=%s fbp=%s event_url=%s", meta.Fbc, meta.Fbp, meta.EventSourceURL)

	var pixelID, capiToken sql.NullString
	err = db.QueryRow(`
		SELECT facebook_pixel_id, facebook_capi_access_token FROM sites WHERE site_id = ?`, siteID).
		Scan(&pixelID, &capiToken)
	if err != nil {
		t.Fatalf("load site: %v", err)
	}
	if !pixelID.Valid || pixelID.String == "" || !capiToken.Valid || capiToken.String == "" {
		t.Skip("site 未配置 pixel/capi token（可执行: UPDATE sites SET facebook_capi_access_token='FAKE_LOCAL_TEST' WHERE site_id=...）")
	}
	if os.Getenv("META_CAPI_SEND") != "1" && !strings.HasPrefix(capiToken.String, "FAKE_") {
		t.Log("META_CAPI_SEND!=1 且非 FAKE_ token，跳过 CAPI 上报")
		return
	}

	tx := &model.PaymentTransaction{
		TransactionID: orderID,
		UserID:        userID,
		SiteID:        siteID,
		Amount:        amount,
		Currency:      currency,
		Snapshot:      snapshot,
		CreatedAt:     createdAt,
	}
	fb := &FacebookClient{apiVersion: "v21.0"}
	err = sendFacebookPurchase(t.Context(), fb, pixelID.String, capiToken.String, os.Getenv("META_TEST_EVENT_CODE"), tx, meta, &u)
	if err != nil {
		t.Logf("CAPI HTTP: %v", err)
	} else {
		t.Log("CAPI Purchase 已发送")
	}
}
