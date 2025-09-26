package orcid

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

const (
	ContentType      = "application/vnd.orcid+xml"
	TokenUrl         = "https://orcid.org/oauth/token"
	SandboxTokenUrl  = "https://sandbox.orcid.org/oauth/token"
	PublicUrl        = "https://pub.orcid.org/v3.0"
	SandboxPublicUrl = "https://pub.sandbox.orcid.org/v3.0"
	MemberUrl        = "https://api.orcid.org/v3.0"
	SandboxMemberUrl = "https://api.sandbox.orcid.org/v3.0"
)

var (
	ErrNotFound  = errors.New("orcid: not found")
	ErrDuplicate = errors.New("orcid: duplicate")

	reID = regexp.MustCompile("^[0-9]{4}-[0-9]{4}-[0-9]{4}-[0-9]{4}$")
)

type Config struct {
	HTTPClient   *http.Client
	ClientID     string
	ClientSecret string
	Scopes       []string
	Token        string
	Sandbox      bool
}

type Client struct {
	httpClient *http.Client
	baseURL    string
}

type MemberClient struct {
	*Client
}

func newClient(baseUrl string, cfg Config) *Client {
	var httpClient *http.Client

	if cfg.HTTPClient != nil {
		httpClient = cfg.HTTPClient
	} else if cfg.Token != "" {
		t := &oauth2.Token{AccessToken: cfg.Token}
		ts := oauth2.StaticTokenSource(t)
		httpClient = oauth2.NewClient(context.Background(), ts)
	} else {
		var tokenUrl string
		if cfg.Sandbox {
			tokenUrl = SandboxTokenUrl
		} else {
			tokenUrl = TokenUrl
		}

		scopes := cfg.Scopes
		if scopes == nil {
			scopes = []string{"/read-public"}
		}

		oauthCfg := clientcredentials.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			TokenURL:     tokenUrl,
			Scopes:       scopes,
		}

		httpClient = oauthCfg.Client(context.Background())
	}

	return &Client{
		httpClient: httpClient,
		baseURL:    baseUrl,
	}
}

func NewClient(cfg Config) *Client {
	if cfg.Sandbox {
		return newClient(SandboxPublicUrl, cfg)
	}
	return newClient(PublicUrl, cfg)
}

func NewMemberClient(cfg Config) *MemberClient {
	if cfg.Sandbox {
		return &MemberClient{newClient(SandboxMemberUrl, cfg)}
	}
	return &MemberClient{newClient(MemberUrl, cfg)}
}

func (c *Client) get(ctx context.Context, path string, params url.Values, resData any) ([]byte, error) {
	return c.request(ctx, "GET", path, params, nil, resData)
}

func (c *Client) request(ctx context.Context, method, path string, params url.Values, reqData, resData any) ([]byte, error) {
	u := c.baseURL + "/" + path

	var buf io.ReadWriter

	if reqData != nil {
		buf = &bytes.Buffer{}
		err := xml.NewEncoder(buf).Encode(reqData)
		if err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, u, buf)
	if err != nil {
		return nil, err
	}

	if params != nil {
		req.URL.RawQuery = params.Encode()
	}

	if reqData != nil {
		req.Header.Set("Content-Type", ContentType)
	}
	req.Header.Set("Accept", ContentType)

	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	b, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("orcid: cannot read response: %w", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 400 {
		return b, fmt.Errorf("orcid: http error %d", res.StatusCode)
	}

	if resData != nil {
		if err = xml.Unmarshal(b, resData); err != nil {
			return b, fmt.Errorf("orcid: cannot decode response: %w", err)
		}
	}

	return b, nil
}

func IsID(id string) bool {
	return reID.MatchString(id)
}
