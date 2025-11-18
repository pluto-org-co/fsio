// Copyright (C) 2025 ZedCloud Org.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

package ioutils

import (
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type RetryTransport struct {
	once        sync.Once
	Parent      http.RoundTripper
	MaxAttempts int
	MinSleep    time.Duration
}

func NewRetryTransport(parent http.RoundTripper, maxAttempts int, minSleep time.Duration) (rt *RetryTransport) {
	return &RetryTransport{
		Parent:      parent,
		MaxAttempts: maxAttempts,
		MinSleep:    minSleep,
	}
}

func (r *RetryTransport) RoundTrip(req *http.Request) (res *http.Response, err error) {
	r.once.Do(func() {
		if r.Parent == nil {
			r.Parent = http.DefaultTransport
		}
	})

	for attempt := range r.MaxAttempts {
		res, err = r.Parent.RoundTrip(req)
		if err == nil {
			return res, nil
		}

		if res == nil {
			return nil, fmt.Errorf("failed to execute request: %w", err)
		}

		if res.StatusCode != http.StatusTooManyRequests {
			return res, fmt.Errorf("failed to execute request: %w", err)
		}

		time.Sleep((1 + time.Duration(attempt)) * r.MinSleep)
	}
	return nil, errors.New("max attempts exceeded")
}

var _ http.RoundTripper = (*RetryTransport)(nil)
