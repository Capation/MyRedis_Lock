package redis_lock

import (
	"context"
	_ "embed"
	"errors"
	"github.com/go-redis/redis/v9"
	"github.com/google/uuid"
	"time"
)

var (
	ErrLockNotHold         = errors.New("未持有锁")
	ErrFailedToPreemptLock = errors.New("加锁失败")
)

type Client struct {
	client redis.Cmdable // redis.Client 单个的redis节点  redis.ClusterClient redis集群
}

func NewClient(c redis.Cmdable) *Client {
	return &Client{
		client: c,
	}
}

// TryLock 加锁操作 返回的是一把锁的实例
func (c *Client) TryLock(ctx context.Context, key string, expiration time.Duration) (*Lock, error) {
	value := uuid.New().String() // 生成一个唯一的value
	res, err := c.client.SetNX(ctx, key, value, expiration).Result()
	//　连接redis超时
	if err != nil {
		return nil, err
	}
	if !res {
		return nil, ErrFailedToPreemptLock
	}
	return NewLock(c.client, key, value), nil
}

type Lock struct {
	client redis.Cmdable
	key    string
	value  string
}

func NewLock(client redis.Cmdable, key string, value string) *Lock {
	return &Lock{
		client: client,
		key:    key,
		value:  value,
	}
}

var (
	//go:embed unlock.lua
	luaUnlock string
)

// UnLock 解锁操作
func (l *Lock) UnLock(ctx context.Context) error {
	// 解锁的时候你要确保，这把锁还是你的锁，没有被别人篡夺
	res, err := l.client.Eval(ctx, luaUnlock, []string{l.key}, l.value).Int64()
	if err == redis.Nil {
		return ErrLockNotHold
	}
	if err != nil {
		return err
	}
	// 要判断 res 是不是1
	if res == 0 {
		// 这把锁不是你的，或者这个key不存在
		return ErrLockNotHold
	}
	return nil
}
