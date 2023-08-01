package middleware

import (
	"context"
	"time"

	"done/tools/errors"
	"done/tools/logging"
	"done/tools/util"

	"github.com/gin-gonic/gin"

	"github.com/patrickmn/go-cache"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

type RateLimiterConfig struct {
	Enable              bool
	AllowedPathPrefixes []string
	SkippedPathPrefixes []string
	Period              int
	MaxRequestsPerIP    int
	MaxRequestsPerUser  int
	StoreType           string // memory
	MemoryStoreConfig   RateLimiterMemoryConfig
}

func RateLimiterWithConfig(config RateLimiterConfig) gin.HandlerFunc {
	if !config.Enable {
		return Empty()
	}

	var store = NewRateLimiterMemoryStore(config.MemoryStoreConfig)

	return func(c *gin.Context) {
		if !AllowedPathPrefixes(c, config.AllowedPathPrefixes...) ||
			SkippedPathPrefixes(c, config.SkippedPathPrefixes...) {
			c.Next()
			return
		}

		var (
			allowed bool
			err     error
		)

		ctx := c.Request.Context()
		if userID := util.FromUserID(ctx); userID != "" {
			allowed, err = store.Allow(ctx, userID, time.Second*time.Duration(config.Period), config.MaxRequestsPerUser)
		} else {
			allowed, err = store.Allow(ctx, c.ClientIP(), time.Second*time.Duration(config.Period), config.MaxRequestsPerIP)
		}

		if err != nil {
			logging.Context(ctx).Error("Rate limiter middleware error", zap.Error(err))
			util.ResError(c, errors.InternalServerError("", "Internal server error, please try again later."))
		} else if allowed {
			c.Next()
		} else {
			util.ResError(c, errors.TooManyRequests("", "Too many requests, please try again later."))
		}
	}
}

type RateLimiterStorer interface {
	Allow(ctx context.Context, identifier string, period time.Duration, maxRequests int) (bool, error)
}

func NewRateLimiterMemoryStore(config RateLimiterMemoryConfig) RateLimiterStorer {
	return &RateLimiterMemoryStore{
		cache: cache.New(config.Expiration, config.CleanupInterval),
	}
}

type RateLimiterMemoryConfig struct {
	Expiration      time.Duration
	CleanupInterval time.Duration
}

type RateLimiterMemoryStore struct {
	cache *cache.Cache
}

func (s *RateLimiterMemoryStore) Allow(ctx context.Context, identifier string, period time.Duration, maxRequests int) (bool, error) {
	if period.Seconds() <= 0 || maxRequests <= 0 {
		return true, nil
	}

	if limiter, exists := s.cache.Get(identifier); exists {
		isAllow := limiter.(*rate.Limiter).Allow()
		s.cache.SetDefault(identifier, limiter)
		return isAllow, nil
	}

	limiter := rate.NewLimiter(rate.Every(period), maxRequests)
	limiter.Allow()
	s.cache.SetDefault(identifier, limiter)

	return true, nil
}
