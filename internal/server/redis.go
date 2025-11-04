package server

import (
	"context"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

func markChunk(ctx context.Context, rdb *redis.Client, fileID string, idx int64) error {
	pipe := rdb.TxPipeline()
	pipe.SAdd(ctx, "upload:"+fileID+":chunks", idx)
	pipe.Expire(ctx, "upload:"+fileID+":chunks", 24*time.Hour)
	_, err := pipe.Exec(ctx)
	return err
}

func listedChunks(ctx context.Context, rdb *redis.Client, fileID string) (map[int64]struct{}, error) {
	members, err := rdb.SMembers(ctx, "upload:"+fileID+":chunks").Result()
	if err != nil {
		return nil, err
	}
	set := make(map[int64]struct{}, len(members))
	for _, m := range members {
		if v, err := strconv.ParseInt(m, 10, 64); err == nil {
			set[v] = struct{}{}
		}
	}
	return set, nil
}

func cleanupChunks(ctx context.Context, rdb *redis.Client, fileID string) error {
	return rdb.Del(ctx, "upload:"+fileID+":chunks").Err()
}