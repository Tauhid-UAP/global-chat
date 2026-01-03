package auth

import (
	"crypto/rand"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"time"
	"context"

	"github.com/google/uuid"

	"github.com/Tauhid-UAP/global-chat/core/redisclient"
)

func randomToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.RawStdEncoding.EncodeToString(b)
}

func CreateSession(ctx context.Context, userID string, ttl time.Duration) (string, string, error) {
	sessionID := uuid.NewString()
	csrfToken := randomToken()

	key := "session:" + sessionID

	err := redisclient.Client.HSet(ctx, key, map[string]interface{}{
		"user_id": userID,
		"csrf_token": csrfToken,
	}).Err()

	if err != nil {
		return "", "", err
	}

	err = redisclient.Client.Expire(ctx, key, ttl).Err()
	if err != nil {
		return "", "", err
	}

	return sessionID, csrfToken, nil
}

func DeleteSession(ctx context.Context, sessionID string) {
	redisclient.Client.Del(ctx, "session:"+sessionID)
}

func GetSession(ctx context.Context, sessionID string) (map[string]string, error) {
	return redisclient.Client.HGetAll(ctx, "session"+sessionID).Result()
}

func Sign(value, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(value))
	return base64.RawStdEncoding.EncodeToString(mac.Sum(nil))
}

func Verify(value, sig, secret string) bool {
	return Sign(value, secret) == sig
}
