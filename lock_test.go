package redis_lock

import (
	"context"
	"errors"
	"github.com/go-redis/redis/v9"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"redis_lock/mocks"
	"testing"
	"time"
)

func TestClient_TryLock(t *testing.T) {

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	// 匿名内部结构体切片
	testCases := []struct {
		name string

		// 设置mock
		mock func() redis.Cmdable

		// 测试的输入
		key        string
		expiration time.Duration

		// 测试的输出
		wantErr  error
		wantLock *Lock
	}{
		// 加锁成功
		{
			name:       "locked",
			key:        "locked-key",
			expiration: time.Minute,
			mock: func() redis.Cmdable {
				rdb := mocks.NewMockCmdable(ctrl)
				res := redis.NewBoolResult(true, nil)
				rdb.EXPECT().
					SetNX(gomock.Any(), "locked-key", gomock.Any(), time.Minute).
					Return(res)
				return rdb
			},

			wantLock: &Lock{
				key: "locked-key",
			},
		},

		// mock 网络错误
		{
			name:       "network error",
			key:        "network-key",
			expiration: time.Minute,
			mock: func() redis.Cmdable {
				rdb := mocks.NewMockCmdable(ctrl)
				res := redis.NewBoolResult(false, errors.New("network error"))
				rdb.EXPECT().
					SetNX(gomock.Any(), "network-key", gomock.Any(), time.Minute).
					Return(res)
				return rdb
			},

			wantErr: errors.New("network error"),
		},

		// 模拟并发竞争失败
		{
			name:       "failed to key",
			key:        "failed-key",
			expiration: time.Minute,
			mock: func() redis.Cmdable {
				rdb := mocks.NewMockCmdable(ctrl)
				res := redis.NewBoolResult(false, nil)
				rdb.EXPECT().
					SetNX(gomock.Any(), "failed-key", gomock.Any(), time.Minute).
					Return(res)
				return rdb
			},

			wantErr: ErrFailedToPreemptLock,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			c := NewClient(tc.mock())
			l, err := c.TryLock(context.Background(), tc.key, tc.expiration)
			assert.Equal(t, tc.wantErr, err)
			if err != nil {
				return
			}
			assert.Equal(t, tc.wantLock.key, l.key)
			assert.NotEmpty(t, l.value)
			assert.NotNil(t, l.client)
		})
	}
}
