package tinkoff

import (
	"github.com/jfk9w-go/tinkoff-api/v2"

	"github.com/jfk9w/hoarder/internal/jobs/tinkoff/internal/loaders"
)

type Client = loaders.Client

type ClientFactory func(params tinkoff.ClientParams) (Client, error)

var defaultClientFactory ClientFactory = func(params tinkoff.ClientParams) (Client, error) {
	return tinkoff.NewClient(params)
}
