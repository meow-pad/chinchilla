package gateway

import (
	"github.com/meow-pad/persian/frame/pboot"
	"github.com/meow-pad/persian/utils/timewheel"
)

type Gateway struct {
	AppInfo  pboot.AppInfo        `autowire:""`
	SecTimer *timewheel.TimeWheel `autowire:"SecondTimer"`

	Options *Options
}
