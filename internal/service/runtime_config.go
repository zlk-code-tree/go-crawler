package service

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/gogf/gf/v2/frame/g"
)

// AppEnv returns the current runtime environment.
// Priority: config app.env > APP_ENV > GO_ENV > GF_ENV > RUN_ENV > dev.
func AppEnv(ctx context.Context) string {
	env := strings.TrimSpace(g.Cfg().MustGet(ctx, "app.env", "").String())
	if env == "" {
		for _, key := range []string{"APP_ENV", "GO_ENV", "GF_ENV", "RUN_ENV"} {
			env = strings.TrimSpace(os.Getenv(key))
			if env != "" {
				break
			}
		}
	}
	if env == "" {
		env = "dev"
	}
	return strings.ToLower(env)
}

func IsProductionEnv(ctx context.Context) bool {
	switch AppEnv(ctx) {
	case "prod", "production":
		return true
	default:
		return false
	}
}

func SchedulerEnabled(ctx context.Context) bool {
	return g.Cfg().MustGet(ctx, "scheduler.enabled", true).Bool()
}

func SchedulerManualTriggerEnabled(ctx context.Context) bool {
	if !SchedulerEnabled(ctx) {
		return false
	}
	return g.Cfg().MustGet(ctx, "scheduler.manualTriggerEnabled", true).Bool()
}

func SchedulerCycleInterval(ctx context.Context) time.Duration {
	minutes := g.Cfg().MustGet(ctx, "scheduler.cycleIntervalMinutes", 180).Int()
	if minutes <= 0 {
		minutes = 180
	}
	return time.Duration(minutes) * time.Minute
}

func SchedulerAutoStartDecision(ctx context.Context) (bool, string) {
	if !SchedulerEnabled(ctx) {
		return false, "scheduler.enabled=false"
	}
	if !g.Cfg().MustGet(ctx, "scheduler.autoStart", false).Bool() {
		return false, "scheduler.autoStart=false"
	}
	if g.Cfg().MustGet(ctx, "scheduler.autoStartOnlyProduction", true).Bool() && !IsProductionEnv(ctx) {
		return false, "非生产环境跳过自动启动"
	}
	return true, "允许自动启动"
}
