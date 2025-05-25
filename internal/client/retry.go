package client

import (
	"context"
	"math"
	"math/rand"
	"net/http"
	"time"
)

func executeWithRetry(ctx context.Context, client *Client, req *http.Request, config *RetryConfig) (*http.Response, error) {
	var (
		resp *http.Response
		err  error
	)

	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		reqClone := cloneRequest(req)

		// Use the client's http.Client directly
		resp, err = client.client.Do(reqClone)

		if !config.RetryPolicy(resp, err) {
			return resp, err
		}

		if attempt == config.MaxRetries {
			return resp, err
		}

		if resp != nil {
			resp.Body.Close()
		}

		var waitTime time.Duration
		if config.Strategy == RetryStrategyExponential {
			waitTime = calculateBackoffDelay(config, attempt)
		} else {
			waitTime = config.FixedDelay
		}

		select {
		case <-ctx.Done():
			if resp != nil {
				resp.Body.Close()
			}
			return nil, ctx.Err()
		case <-time.After(waitTime):
		}
	}

	return resp, err
}

func calculateBackoffDelay(config *RetryConfig, attempt int) time.Duration {
	if config == nil || config.Strategy != RetryStrategyExponential {
		return 0
	}

	delay := time.Duration(float64(config.InitialDelay) * math.Pow(config.Multiplier, float64(attempt)))

	if delay > config.MaxDelay {
		delay = config.MaxDelay
	}

	if config.Jitter {
		jitterRange := float64(delay) * 0.25
		jitter := time.Duration(rand.Float64() * jitterRange)
		delay += jitter
	}

	return delay
}

func cloneRequest(req *http.Request) *http.Request {
	clone := req.Clone(req.Context())

	if req.Body == nil {
		return clone
	}

	return clone
}
