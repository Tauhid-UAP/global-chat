package redisclient

import (
	"context"
	"os"
	"strconv"

	"github.com/redis/go-redis/v9"

	"github.com/Tauhid-UAP/global-chat/core/models"
)

var Client *redis.Client

func AddUser(ctx context.Context, cacheKey string, userID string, firstName string, lastName string, isAnonymous bool) error {
	return Client.HSet(ctx, cacheKey, map[string]interface{}{
		"id": userID,
		"first_name": firstName,
		"last_name": lastName,
		"is_anonymous": isAnonymous,
	}).Err()
}

func GetUserByCacheKey(ctx context.Context, cacheKey string) (models.User, error) {
	data, err := Client.HGetAll(ctx, cacheKey).Result()
	if err != nil {
		return models.User{}, err
	}
	
	isAnonymousUser, err := strconv.ParseBool(data["is_anonymous"])
	if err != nil {
		return models.User{}, err
	}

	return models.User {
		ID: data["id"],
		FirstName: data["first_name"],
		LastName: data["last_name"],
		IsAnonymous: isAnonymousUser,
	}, nil
}

var PubSubClient *redis.Client

func Init() {
	Client = redis.NewClient(&redis.Options{
		Addr: os.Getenv("REDIS_ADDR"),
		Password: os.Getenv("REDIS_PASSWORD"),
		DB: 0,
	})

	PubSubClient = redis.NewClient(&redis.Options{
		Addr: os.Getenv("REDIS_PUB_SUB_ADDR"),
		Password: os.Getenv("REDIS_PUB_SUB_PASSWORD"),
		DB: 0,
	})
}

func PublishToRoom(ctx context.Context, room string, payload []byte) error {
	return PubSubClient.Publish(ctx, room, payload).Err()
}

func SubscribeToRoom(ctx context.Context, room string) *redis.PubSub {
	return PubSubClient.Subscribe(ctx, room)
}

func Ping(ctx context.Context) error {
	return Client.Ping(ctx).Err()
}

func PingPubSub(ctx context.Context) error {
	return PubSubClient.Ping(ctx).Err()
}
