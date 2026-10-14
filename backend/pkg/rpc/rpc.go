// Package distributed contains explicit inter-service contracts; it never opens another service's database.
package rpc

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const Prefix = "/internal/v1/"

type RPCError struct{ Status int }

func (e *RPCError) Error() string { return fmt.Sprintf("service returned status %d", e.Status) }

func AuthFailureStatus(err error) int {
	var response *RPCError
	if errors.As(err, &response) && response.Status == http.StatusUnauthorized {
		return http.StatusUnauthorized
	}
	return http.StatusServiceUnavailable
}

type Client struct {
	URL, Token string
	HTTP       *http.Client
}

func NewClient(address, token string) (*Client, error) {
	u, err := url.Parse(address)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") {
		return nil, errors.New("invalid service origin")
	}
	if len(token) < 32 {
		return nil, errors.New("service token requires at least 32 bytes")
	}
	return &Client{URL: strings.TrimRight(address, "/"), Token: token, HTTP: &http.Client{Timeout: 10 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}, nil
}
func (c *Client) Call(ctx context.Context, op string, input, output any) error {
	data, err := json.Marshal(input)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, "POST", c.URL+Prefix+op, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Content-Type", "application/json")
	res, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return &RPCError{Status: res.StatusCode}
	}
	if output == nil {
		_, err = io.Copy(io.Discard, io.LimitReader(res.Body, 1<<20))
		return err
	}
	return json.NewDecoder(io.LimitReader(res.Body, 4<<20)).Decode(output)
}
func Protect(token string) gin.HandlerFunc {
	expected := sha256.Sum256([]byte("Bearer " + token))
	return func(c *gin.Context) {
		actual := sha256.Sum256([]byte(c.GetHeader("Authorization")))
		if len(token) < 32 || subtle.ConstantTimeCompare(expected[:], actual[:]) != 1 {
			c.AbortWithStatus(401)
			return
		}
		c.Next()
	}
}
func Bind(c *gin.Context, out any) bool {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 1<<20)
	d := json.NewDecoder(c.Request.Body)
	d.DisallowUnknownFields()
	if d.Decode(out) != nil || d.Decode(new(any)) != io.EOF {
		c.AbortWithStatus(400)
		return false
	}
	return true
}
