package api

import (
	"errors"
	"time"
)

// sendMailWithTimeout runs fn and returns error if it doesn't complete within timeout.
// It does not forcibly cancel the underlying network dial; it's a soft timeout suitable for admin/test flows.
func sendMailWithTimeout(timeout time.Duration, fn func() error) error {
	done := make(chan error, 1)
	go func() { done <- fn() }()
	select {
	case err := <-done:
		return err
	case <-time.After(timeout):
		return errors.New("smtp send timed out")
	}
}
