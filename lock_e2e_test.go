package redis_lock

import (
	"context"
	"github.com/go-redis/redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestClient_TryLock_e2e(t *testing.T) {

	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379", // use default Addr
		Password: "",               // no password set
		DB:       0,                // use default DB
	})

	testCases := []struct {
		name string

		// 准备数据
		before func()
		// 校验redis数据并清洗数据
		after func()

		// 测试的输入
		key        string
		expiration time.Duration

		// 测试的输出
		wantErr  error
		wantLock *Lock
	}{
		{
			name:   "locked",
			before: func() {},
			after: func() {
				// 校验数据
				res, err := rdb.Del(context.Background(), "locked-key").Result()
				require.NoError(t, err)
				require.Equal(t, int64(1), res)
			},
			key:        "locked-key",
			expiration: time.Minute,
			wantLock: &Lock{
				key: "locked-key",
			},
		},

		{
			name: "failed to lock",
			key:  "failed-key",
			before: func() {
				val, err := rdb.Set(context.Background(), "failed-key", "123", time.Minute).Result()
				require.NoError(t, err)
				require.Equal(t, "OK", val)
			},
			after: func() {
				// 校验数据
				res, err := rdb.Get(context.Background(), "failed-key").Result()
				require.NoError(t, err)
				require.Equal(t, "123", res)
			},
			expiration: time.Minute,
			wantErr:    ErrFailedToPreemptLock,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.before()
			c := NewClient(rdb)
			l, err := c.TryLock(context.Background(), tc.key, tc.expiration)
			assert.Equal(t, tc.wantErr, err)
			if err != nil {
				return
			}
			tc.after()
			assert.Equal(t, tc.wantLock.key, l.key)
			assert.NotEmpty(t, l.value)
			assert.NotNil(t, l.client)
		})
	}
}
