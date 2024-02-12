package ratelimitter

import (
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func CombinedRateLimiter() echo.MiddlewareFunc {
	config1 := middleware.RateLimiterConfig{
		Skipper: middleware.DefaultSkipper,
		Store: middleware.NewRateLimiterMemoryStoreWithConfig(
			middleware.RateLimiterMemoryStoreConfig{
				Rate:      2.0 / 60,
				Burst:     2,
				ExpiresIn: 60 * time.Second, 
			},
		),
		IdentifierExtractor: func(ctx echo.Context) (string, error) {
			id := ctx.RealIP()
			fmt.Println("IP Address", id)
			return id, nil
		},
		ErrorHandler: func(context echo.Context, err error) error {
			return context.JSON(http.StatusForbidden, "errorhandler")
		},
		DenyHandler: func(context echo.Context, identifier string, err error) error {
			return context.JSON(http.StatusTooManyRequests, "Limit Exceeded")
		},
	}

	config2 := middleware.RateLimiterConfig{
		Skipper: middleware.DefaultSkipper,
		Store: middleware.NewRateLimiterMemoryStoreWithConfig(
			middleware.RateLimiterMemoryStoreConfig{
				Rate:      20.0 / (24 * 60 * 60),
				Burst:     20,
				ExpiresIn: 24 * time.Hour,
			},
		),
		IdentifierExtractor: func(ctx echo.Context) (string, error) {
			id := ctx.RealIP()
			return id, nil
		},
		ErrorHandler: func(context echo.Context, err error) error {
			return context.JSON(http.StatusForbidden, "errorhandler")
		},
		DenyHandler: func(context echo.Context, identifier string, err error) error {
			return context.JSON(http.StatusTooManyRequests, "Limit Exceeded per day")
		},
	}
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			err := middleware.RateLimiterWithConfig(config1)(func(c echo.Context) error {
				err := middleware.RateLimiterWithConfig(config2)(next)(c)
				if err != nil {
					return c.JSON(http.StatusTooManyRequests, "Limit Exceeded per day")
				}
				return nil
			})(c)
			if err != nil {
				return c.JSON(http.StatusTooManyRequests, "Limit Exceeded per minute")
			}

			return nil
		}
	}
}