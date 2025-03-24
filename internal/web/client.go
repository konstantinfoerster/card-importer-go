package web

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"slices"
	"sync/atomic"
	"time"

	"github.com/konstantinfoerster/card-importer-go/internal/aio"
	"github.com/rs/zerolog/log"
)

type Config struct {
	Delay       time.Duration `yaml:"delay"`
	Timeout     time.Duration `yaml:"timeout"`
	Retries     int32         `yaml:"retries"`
	Retrieables []int         `yaml:"retrieables"`
	RetryDelay  time.Duration `yaml:"retryDelay"`
}

type Response struct {
	Body     io.ReadCloser
	MimeType MimeType
}

func NewGetOpts() GetOptions {
	return GetOptions{
		Header:      make(map[string]string),
		StatusCodes: []int{http.StatusOK},
	}
}

type GetOptions struct {
	Header      map[string]string
	StatusCodes []int
}

func (o GetOptions) WithHeader(k, v string) GetOptions {
	o.Header[k] = v

	return o
}

func (o GetOptions) WithExpectedCodes(statusCode ...int) GetOptions {
	o.StatusCodes = statusCode

	return o
}

type Client interface {
	Get(ctx context.Context, url string, opts GetOptions) (*Response, error)
}

func NewClient(cfg Config, client *http.Client) Client {
	if client == nil {
		panic("missing net/http client")
	}

	return &httpClient{
		cfg:    cfg,
		client: client,
	}
}

type httpClient struct {
	cfg    Config
	client *http.Client
}

func (c *httpClient) Get(ctx context.Context, url string, opts GetOptions) (*Response, error) {
	return WithRetry(ctx, c.cfg, func() (*http.Response, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, fmt.Errorf("request creation failed for url %s, %w", url, err)
		}

		req.Header.Set(HeaderUserAgent, DefaultUserAgent)
		for k, v := range opts.Header {
			req.Header.Set(k, v)
		}

		resp, err := c.client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("request execution failed for url %s, %w", url, err)
		}

		if !slices.Contains(opts.StatusCodes, resp.StatusCode) {
			defer aio.Close(resp.Body)

			return nil, NewHTTPErr(url, resp)
		}

		return resp, nil
	})
}

func WithRetry(ctx context.Context, cfg Config, exec func() (*http.Response, error)) (*Response, error) {
	t := time.NewTimer(cfg.Delay)
	defer t.Stop()

	var counter atomic.Int32
	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("stop execution due to cancelled context")
		case <-t.C:
			resp, err := exec()
			if err != nil {
				if resp != nil {
					aio.Close(resp.Body)
				}

				if IsStatusCode(err, cfg.Retrieables...) {
					if counter.Load() == cfg.Retries {
						return nil, err
					}

					log.Info().Str("err", err.Error()).Msgf("request attempt %d after err", counter.Load()+1)
					counter.Add(1)

					t.Reset(cfg.RetryDelay)

					continue
				}

				return nil, err
			}

			return &Response{
				Body:     resp.Body,
				MimeType: NewMimeType(resp.Header.Get("content-type")),
			}, nil
		}
	}
}
