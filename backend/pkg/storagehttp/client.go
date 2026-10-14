package storagehttp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	ports "go-cloud-storage/backend/pkg/ports"
	"go-cloud-storage/backend/pkg/storagekey"
)

type Client struct {
	endpoint string
	token    string
	http     *http.Client
}

func NewClient(endpoint, token string) (*Client, error) {
	u, err := url.Parse(endpoint)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") {
		return nil, errors.New("storage endpoint must be an http(s) origin without credentials, path or query")
	}
	if len(token) < 32 {
		return nil, errors.New("storage service token must contain at least 32 bytes")
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.ResponseHeaderTimeout = 5 * time.Minute
	return &Client{endpoint: strings.TrimRight(endpoint, "/"), token: token, http: &http.Client{
		Transport: transport, Timeout: 30 * time.Minute,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}}, nil
}

func (c *Client) do(ctx context.Context, method, path string, query url.Values, body io.Reader, size int64, contentType string) (*http.Response, error) {
	target := c.endpoint + Prefix + path
	if len(query) > 0 {
		target += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, method, target, body)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.ContentLength = size
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", contentType)
	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("storage request failed: %w", err)
	}
	if res.StatusCode != http.StatusOK {
		res.Body.Close()
		return nil, fmt.Errorf("storage request failed with status %d", res.StatusCode)
	}
	return res, nil
}

func decode(res *http.Response, output any) error {
	defer res.Body.Close()
	return json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(output)
}

func (c *Client) rpc(ctx context.Context, operation string, input Request) (Response, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	data, err := json.Marshal(input)
	if err != nil {
		return Response{}, err
	}
	res, err := c.do(ctx, http.MethodPost, operation, nil, bytes.NewReader(data), int64(len(data)), "application/json")
	if err != nil {
		return Response{}, err
	}
	var output Response
	err = decode(res, &output)
	return output, err
}

func (c *Client) GenerateObjectKey(userID int, parentID, name string) string {
	return storagekey.New(userID, parentID, name)
}
func (c *Client) ThumbnailObjectKey(key string) string { return storagekey.Thumbnail(key) }
func (c *Client) GenerateObjectURL(key string) string {
	out, err := c.rpc(context.Background(), "url", Request{Key: key})
	if err != nil {
		return ""
	}
	return out.Value
}
func (c *Client) UploadFromStream(ctx context.Context, userID int, r io.Reader, name string, size int64, hash, parentID string) (*ports.StoredObject, error) {
	key := c.GenerateObjectKey(userID, parentID, name)
	out, err := c.put(ctx, key, r, size, "application/octet-stream")
	if err != nil {
		return nil, err
	}
	if out.Object == nil {
		return nil, errors.New("storage returned no uploaded object")
	}
	return out.Object, nil
}
func (c *Client) put(ctx context.Context, key string, r io.Reader, size int64, contentType string) (Response, error) {
	res, err := c.do(ctx, http.MethodPut, "object", url.Values{"key": {key}}, r, size, contentType)
	if err != nil {
		return Response{}, err
	}
	var out Response
	err = decode(res, &out)
	return out, err
}
func (c *Client) PutObjectStream(ctx context.Context, key string, r io.Reader, size int64, contentType string) (int64, error) {
	out, err := c.put(ctx, key, r, size, contentType)
	return out.Size, err
}
func (c *Client) InitiateMultipartUpload(ctx context.Context, key string) (string, error) {
	out, err := c.rpc(ctx, "multipart/init", Request{Key: key})
	return out.Value, err
}
func (c *Client) UploadPart(ctx context.Context, key, uploadID string, part int, r io.Reader, size int64, hash string) (*ports.UploadedPart, string, error) {
	res, err := c.do(ctx, http.MethodPut, "part", url.Values{"key": {key}, "uploadId": {uploadID}, "part": {strconv.Itoa(part)}, "hash": {hash}}, r, size, "application/octet-stream")
	if err != nil {
		return nil, "", err
	}
	var out Response
	if err = decode(res, &out); err != nil {
		return nil, "", err
	}
	if out.Part == nil {
		return nil, "", errors.New("storage returned no uploaded part")
	}
	return out.Part, out.Hash, nil
}
func (c *Client) CompleteMultipartUpload(ctx context.Context, key, id string, parts []ports.CompletedPart) (*ports.StoredObject, error) {
	out, err := c.rpc(ctx, "multipart/complete", Request{Key: key, UploadID: id, Parts: parts})
	if err == nil && out.Object == nil {
		err = errors.New("storage returned no completed object")
	}
	return out.Object, err
}
func (c *Client) AbortMultipartUpload(ctx context.Context, key, id string) error {
	_, err := c.rpc(ctx, "multipart/abort", Request{Key: key, UploadID: id})
	return err
}
func (c *Client) GetObjectInfo(ctx context.Context, key string) (int64, error) {
	out, err := c.rpc(ctx, "stat", Request{Key: key})
	return out.Size, err
}
func (c *Client) ComputeObjectSHA256(ctx context.Context, key string) (string, error) {
	out, err := c.rpc(ctx, "hash", Request{Key: key})
	return out.Value, err
}
func (c *Client) DownloadFile(ctx context.Context, key string) (io.ReadCloser, error) {
	res, err := c.do(ctx, http.MethodGet, "object", url.Values{"key": {key}}, nil, 0, "")
	if err != nil {
		return nil, err
	}
	return res.Body, nil
}
func (c *Client) DownloadFileRange(ctx context.Context, key string, start, end int64) (io.ReadCloser, error) {
	res, err := c.do(ctx, http.MethodGet, "object", url.Values{"key": {key}, "start": {strconv.FormatInt(start, 10)}, "end": {strconv.FormatInt(end, 10)}}, nil, 0, "")
	if err != nil {
		return nil, err
	}
	return res.Body, nil
}
func (c *Client) PresignedGetPreviewURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	out, err := c.rpc(ctx, "presign/preview", Request{Key: key, Expiry: expiry})
	return out.Value, err
}
func (c *Client) PresignedDownloadURL(ctx context.Context, key, name string, expiry time.Duration) (string, error) {
	out, err := c.rpc(ctx, "presign/download", Request{Key: key, FileName: name, Expiry: expiry})
	return out.Value, err
}
func (c *Client) PresignStoredObjectURL(ctx context.Context, storedURL string, expiry time.Duration) string {
	if storedURL == "" {
		return ""
	}
	out, err := c.rpc(ctx, "presign/stored", Request{StoredURL: storedURL, Expiry: expiry})
	if err != nil {
		return ""
	}
	return out.Value
}
func (c *Client) DeleteFile(ctx context.Context, key string) error {
	_, err := c.rpc(ctx, "delete", Request{Key: key})
	return err
}
func (c *Client) DeleteObjectWithThumbnail(ctx context.Context, key string) error {
	_, err := c.rpc(ctx, "delete-with-thumbnail", Request{Key: key})
	return err
}
func (c *Client) DeleteThumbnailForObject(ctx context.Context, key string) error {
	_, err := c.rpc(ctx, "delete-thumbnail", Request{Key: key})
	return err
}
func (c *Client) DeleteFiles(ctx context.Context, keys []string) error {
	_, err := c.rpc(ctx, "delete-many", Request{Keys: keys})
	return err
}
func (c *Client) CopyObject(ctx context.Context, source, destination string) error {
	_, err := c.rpc(ctx, "copy", Request{Key: source, Destination: destination})
	return err
}
func (c *Client) GenerateThumbnailForObject(ctx context.Context, key string) (string, error) {
	out, err := c.rpc(ctx, "thumbnail", Request{Key: key})
	return out.Value, err
}

var _ ports.Storage = (*Client)(nil)
