package tinkoff

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/go-querystring/query"
	"github.com/jfk9w-go/based"
	"github.com/pkg/errors"
)

const (
	baseURL      = "https://www.tbank.ru"
	baseApiURL   = baseURL + "/api"
	pingInterval = time.Minute
)

var (
	ErrNoDataFound        = errors.New("no data found")
	errMaxRetriesExceeded = errors.New("max retries exceeded")
	errUnauthorized       = errors.New("no sessionid")
)

type Session struct {
	ID string
}

type SessionStorage interface {
	LoadSession(ctx context.Context, phone string) (*Session, error)
	UpdateSession(ctx context.Context, phone string, session *Session) error
}

type Credential struct {
	Phone    string
	Password string
}

type ClientParams struct {
	Clock          based.Clock    `validate:"required"`
	Credential     Credential     `validate:"required"`
	SessionStorage SessionStorage `validate:"required"`

	AuthFlow  AuthFlow
	Transport http.RoundTripper
}

type Client struct {
	httpClient   *http.Client
	authFlow     AuthFlow
	credential   Credential
	session      *based.WriteThroughCached[*Session]
	rateLimiters map[string]based.Locker
	mu           based.RWMutex
}

func NewClient(params ClientParams) (*Client, error) {
	if err := based.Validate(params); err != nil {
		return nil, err
	}

	authFlow := params.AuthFlow
	if authFlow == nil {
		authFlow = ApiAuthFlow
	}

	return &Client{
		httpClient: &http.Client{
			Transport: params.Transport,
		},
		authFlow:   authFlow,
		credential: params.Credential,
		session: based.NewWriteThroughCached(
			based.WriteThroughCacheStorageFunc[string, *Session]{
				LoadFn:   params.SessionStorage.LoadSession,
				UpdateFn: params.SessionStorage.UpdateSession,
			},
			params.Credential.Phone,
		),
		rateLimiters: map[string]based.Locker{
			shoppingReceiptPath: based.Lockers{
				based.Semaphore(params.Clock, 25, 75*time.Second),
				based.Semaphore(params.Clock, 75, 11*time.Minute),
			},
		},
	}, nil
}

func (c *Client) Ping(ctx context.Context) {
	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()
	for {
		_ = c.ping(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (c *Client) AccountsLightIb(ctx context.Context) (AccountsLightIbOut, error) {
	resp, err := executeCommon(ctx, c, accountsLightIbIn{})
	if err != nil {
		return nil, err
	}

	return resp.Payload, nil
}

func (c *Client) Statements(ctx context.Context, in *StatementsIn) (StatementsOut, error) {
	resp, err := executeCommon(ctx, c, in)
	if err != nil {
		return nil, err
	}

	return resp.Payload, nil
}

func (c *Client) AccountRequisites(ctx context.Context, in *AccountRequisitesIn) (*AccountRequisitesOut, error) {
	resp, err := executeCommon(ctx, c, in)
	if err != nil {
		return nil, err
	}

	return resp.Payload, nil
}

func (c *Client) Operations(ctx context.Context, in *OperationsIn) (OperationsOut, error) {
	resp, err := executeCommon(ctx, c, in)
	if err != nil {
		return nil, err
	}

	return resp.Payload, nil
}

func (c *Client) ShoppingReceipt(ctx context.Context, in *ShoppingReceiptIn) (*ShoppingReceiptOut, error) {
	resp, err := executeCommon(ctx, c, in)
	if err != nil {
		return nil, err
	}

	return &resp.Payload, nil
}

func (c *Client) ClientOfferEssences(ctx context.Context) (ClientOfferEssencesOut, error) {
	resp, err := executeCommon(ctx, c, clientOfferEssencesIn{})
	if err != nil {
		return nil, err
	}

	return resp.Payload, nil
}

func (c *Client) InvestOperationTypes(ctx context.Context) (*InvestOperationTypesOut, error) {
	return executeInvest(ctx, c, investOperationTypesIn{})
}

func (c *Client) InvestAccounts(ctx context.Context, in *InvestAccountsIn) (*InvestAccountsOut, error) {
	return executeInvest(ctx, c, in)
}

func (c *Client) InvestOperations(ctx context.Context, in *InvestOperationsIn) (*InvestOperationsOut, error) {
	return executeInvest(ctx, c, in)
}

func (c *Client) InvestCandles(ctx context.Context, in *InvestCandlesIn) (*InvestCandlesOut, error) {
	resp, err := executeCommon(ctx, c, in)
	if err != nil {
		return nil, err
	}

	return &resp.Payload, nil
}

func (c *Client) rateLimiter(path string) based.Locker {
	if rateLimiter, ok := c.rateLimiters[path]; ok {
		return rateLimiter
	}

	return based.Unlocker
}

func (c *Client) getSessionID(ctx context.Context) (string, error) {
	session, err := c.session.Get(ctx)
	if err != nil {
		return "", errors.Wrap(err, "get sessionid")
	}

	if session == nil {
		return "", errUnauthorized
	}

	return session.ID, nil
}

func (c *Client) ensureSessionID(ctx context.Context) (string, error) {
	var err error

	session := getSession(ctx)
	if session == nil {
		session, err = c.session.Get(ctx)
		if err != nil {
			return "", err
		}
	}

	if session == nil {
		if session, err = c.authorize(ctx); err != nil {
			return "", errors.Wrap(err, "authorize")
		}

		if err = c.session.Update(ctx, session); err != nil {
			return "", errors.Wrap(err, "store new sessionid")
		}
	}

	return session.ID, nil
}

func (c *Client) resetSessionID(ctx context.Context) error {
	return c.session.Update(ctx, nil)
}

func (c *Client) authorize(ctx context.Context) (*Session, error) {
	authorizer := getAuthorizer(ctx)
	if authorizer == nil {
		return nil, errors.New("authorizer is required, but not set")
	}

	return c.authFlow.authorize(ctx, c, authorizer)
}

func (c *Client) ping(ctx context.Context) error {
	ctx, cancel := c.mu.Lock(ctx)
	defer cancel()
	if err := ctx.Err(); err != nil {
		return err
	}

	out, err := executeCommon(ctx, c, pingIn{})
	if err != nil {
		return errors.Wrap(err, "ping")
	}

	if out.Payload.AccessLevel != "CLIENT" {
		if err := c.resetSessionID(ctx); err != nil {
			return errors.Wrap(err, "reset sessionid")
		}

		return errUnauthorized
	}

	return nil
}

func executeInvest[R any](ctx context.Context, c *Client, in investExchange[R]) (*R, error) {
	ctx, cancel := c.rateLimiter(in.path()).Lock(ctx)
	defer cancel()
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	var sessionID string
	if in.auth() {
		ctx, cancel = c.mu.Lock(ctx)
		defer cancel()
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		var err error
		sessionID, err = c.ensureSessionID(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "ensure sessionid")
		}
	}

	urlQuery, err := query.Values(in)
	if err != nil {
		return nil, errors.Wrap(err, "encode url query")
	}

	if sessionID != "" {
		urlQuery.Set("sessionId", sessionID)
	}

	httpReq, err := http.NewRequest(http.MethodGet, baseApiURL+in.path(), nil)
	if err != nil {
		return nil, errors.Wrap(err, "create http request")
	}

	httpReq.URL.RawQuery = urlQuery.Encode()
	httpReq.Header.Set("X-App-Name", "invest")
	httpReq.Header.Set("X-App-Version", "1.328.0")

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, errors.Wrap(err, "execute request")
	}

	if httpResp.Body == nil {
		return nil, errors.New(httpResp.Status)
	}

	defer httpResp.Body.Close()

	switch {
	case httpResp.StatusCode == http.StatusOK:
		var resp R
		if err := json.NewDecoder(httpResp.Body).Decode(&resp); err != nil {
			return nil, errors.Wrap(err, "unmarshal response body")
		}

		return &resp, nil

	case httpResp.StatusCode >= 400 && httpResp.StatusCode < 600:
		var investErr investError
		if body, err := io.ReadAll(httpResp.Body); err != nil {
			return nil, errors.New(httpResp.Status)
		} else if err := json.Unmarshal(body, &investErr); err != nil {
			return nil, errors.New(ellipsis(body))
		} else {
			if investErr.ErrorCode == "404" || investErr.ErrorCode == "Forbidden" {
				// this may be due to expired sessionid, try to check it
				if err := c.ping(ctx); errors.Is(err, errUnauthorized) {
					retry := &retryStrategy{
						timeout:    constantRetryTimeout(0),
						maxRetries: 1,
					}

					ctx, err := retry.do(ctx)
					if err != nil {
						return nil, investErr
					}

					if _, err := c.authorize(ctx); err != nil {
						return nil, errors.Wrap(err, "authorize")
					}

					return executeInvest(ctx, c, in)
				}
			}

			return nil, investErr
		}

	default:
		_, _ = io.Copy(io.Discard, httpResp.Body)
		return nil, errors.New(httpResp.Status)
	}
}

func ellipsis(data []byte) string {
	str := string(data)
	if len(str) > 200 {
		return str + "..."
	}

	return str
}

func executeCommon[R any](ctx context.Context, c *Client, in commonExchange[R]) (*commonResponse[R], error) {
	ctx, cancel := c.rateLimiter(in.path()).Lock(ctx)
	defer cancel()
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	var sessionID string
	if in.auth() != none {
		var (
			cancel context.CancelFunc
			err    error
		)

		ctx, cancel = c.mu.Lock(ctx)
		defer cancel()
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		switch in.auth() {
		case force:
			sessionID, err = c.ensureSessionID(ctx)
		case check:
			sessionID, err = c.getSessionID(ctx)
		default:
			return nil, errors.Errorf("unsupported auth %v", in.auth())
		}

		if err != nil {
			return nil, errors.Wrap(err, "get sessionid")
		}
	}

	reqBody, err := query.Values(in)
	if err != nil {
		return nil, errors.Wrap(err, "encode form values")
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, baseApiURL+in.path(), strings.NewReader(reqBody.Encode()))
	if err != nil {
		return nil, errors.Wrap(err, "create request")
	}

	urlQuery := make(url.Values)
	urlQuery.Set("origin", "web,ib5,platform")
	if sessionID != "" {
		urlQuery.Set("sessionid", sessionID)
	}

	httpReq.URL.RawQuery = urlQuery.Encode()
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, errors.Wrap(err, "execute request")
	}

	if httpResp.Body == nil {
		return nil, errors.New(httpResp.Status)
	}

	defer httpResp.Body.Close()

	var (
		respErr error
		retry   *retryStrategy
	)

	if httpResp.StatusCode != http.StatusOK {
		if body, err := io.ReadAll(httpResp.Body); err != nil {
			respErr = errors.New(httpResp.Status)
		} else {
			respErr = errors.New(ellipsis(body))
		}

		retry = &retryStrategy{
			timeout:    exponentialRetryTimeout(time.Second, 2, 0.5),
			maxRetries: -1,
		}
	} else {
		var resp commonResponse[R]
		if err := json.NewDecoder(httpResp.Body).Decode(&resp); err != nil {
			return nil, errors.Wrap(err, "decode response body")
		}

		if in.exprc() == resp.ResultCode {
			return &resp, nil
		}

		respErr = resultCodeError{
			actual:   resp.ResultCode,
			expected: in.exprc(),
			message:  resp.ErrorMessage,
		}

		switch resp.ResultCode {
		case "NO_DATA_FOUND":
			return nil, ErrNoDataFound

		case "REQUEST_RATE_LIMIT_EXCEEDED":
			retry = &retryStrategy{
				timeout:    exponentialRetryTimeout(time.Minute, 2, 0.2),
				maxRetries: 5,
			}

		case "INSUFFICIENT_PRIVILEGES":
			if _, err := c.authorize(ctx); err != nil {
				return nil, errors.Wrap(err, "authorize")
			}

			retry = &retryStrategy{
				timeout:    constantRetryTimeout(0),
				maxRetries: 1,
			}
		}
	}

	if retry != nil {
		ctx, retryErr := retry.do(ctx)
		switch {
		case errors.Is(retryErr, errMaxRetriesExceeded):
			// fallthrough
		case retryErr != nil:
			return nil, retryErr
		default:
			return executeCommon(ctx, c, in)
		}
	}

	return nil, respErr
}
