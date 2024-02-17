package firefly

import (
	"net/http"

	ht "github.com/ogen-go/ogen/http"
)

type httpClient struct {
	client ht.Client
}

func (c httpClient) Do(req *http.Request) (*http.Response, error) {
	req.Header.Set("Accept", "application/json")
	return c.client.Do(req)
}
