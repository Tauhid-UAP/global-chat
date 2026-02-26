package userselector

import (
	"context"
	
	"github.com/Tauhid-UAP/global-chat/core/models"
	"github.com/Tauhid-UAP/global-chat/core/store"
	"github.com/Tauhid-UAP/global-chat/core/redisclient"
)

func GetUserByIDFromApplicableStore(ctx context.Context, userID string, isAnonymousUser bool) (models.User, error) {
	if isAnonymousUser {
		return redisclient.GetUserByCacheKey(ctx, "anonymous_user:" + userID)
	}

	return store.GetUserByID(ctx, userID)
}
