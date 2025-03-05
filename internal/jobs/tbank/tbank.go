package tbank

import (
	tbank "github.com/jfk9w-go/tbank-api"

	"github.com/jfk9w/hoarder/internal/jobs/tbank/internal/loaders"
)

type Client = loaders.Client

type ClientFactory func(params tbank.ClientParams) (Client, error)

var defaultClientFactory ClientFactory = func(params tbank.ClientParams) (Client, error) {
	return tbank.NewClient(params)
}
