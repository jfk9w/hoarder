package lkdr

import (
	"github.com/jfk9w-go/based"
	"github.com/jfk9w/hoarder/internal/captcha"
)

type JobParams struct {
	Config        Config      `validate:"required"`
	Clock         based.Clock `validate:"required"`
	ClientFactory ClientFactory
	CaptchaSolver captcha.TokenProvider
}

type Job struct {
}
