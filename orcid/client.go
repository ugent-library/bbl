package orcid

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
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

func (c *Client) get(path string, data any) (*http.Response, error) {
	req, err := c.newRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}
	res, err := c.do(req, data)
	if err != nil {
		return res, err
	}
	if res.StatusCode == 404 {
		err = ErrNotFound
	} else if res.StatusCode != 200 {
		err = fmt.Errorf("orcid: couldn't get %s", path)
	}
	return res, err
}

// func (c *MemberClient) add(path string, body any) (int, *http.Response, error) {
// 	req, err := c.newRequest("POST", path, body)
// 	if err != nil {
// 		return 0, nil, err
// 	}
// 	res, err := c.do(req, nil)
// 	if err != nil {
// 		return 0, res, err
// 	}
// 	if res.StatusCode == 409 {
// 		return 0, res, ErrDuplicate
// 	}
// 	if res.StatusCode != 201 {
// 		err = fmt.Errorf("couldn't add %s", path)
// 		return 0, res, err
// 	}
// 	loc, err := res.Location()
// 	if err != nil {
// 		return 0, res, err
// 	}
// 	r := regexp.MustCompile("([^/]+)$")
// 	match := r.FindString(loc.String())
// 	putCode, err := strconv.Atoi(match)
// 	return putCode, res, err
// }

// func (c *MemberClient) update(path string, body, data any) (*http.Response, error) {
// 	req, err := c.newRequest("PUT", path, body)
// 	if err != nil {
// 		return nil, err
// 	}
// 	res, err := c.do(req, data)
// 	if err != nil {
// 		return res, err
// 	}
// 	if res.StatusCode != 200 {
// 		err = fmt.Errorf("couldn't update %s", path)
// 		return res, err
// 	}
// 	return res, err
// }

// func (c *Client) delete(path string) (bool, *http.Response, error) {
// 	var ok bool
// 	req, err := c.newRequest("DELETE", path, nil)
// 	if err != nil {
// 		return ok, nil, err
// 	}
// 	res, err := c.do(req, nil)
// 	if err != nil {
// 		return false, res, err
// 	}
// 	if res.StatusCode == 204 {
// 		ok = true
// 	}
// 	return ok, res, err
// }

func (c *Client) newRequest(method, path string, body any) (*http.Request, error) {
	u := c.baseURL + "/" + path
	var buf io.ReadWriter
	if body != nil {
		buf = &bytes.Buffer{}
		err := xml.NewEncoder(buf).Encode(body)
		if err != nil {
			return nil, err
		}
	}
	req, err := http.NewRequest(method, u, buf)
	if err != nil {
		return req, err
	}
	if body != nil {
		req.Header.Set("Content-Type", ContentType)
	}
	req.Header.Set("Accept", ContentType)

	return req, nil
}

func (c *Client) do(req *http.Request, data any) (*http.Response, error) {
	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	if data != nil {
		b, err := io.ReadAll(res.Body)
		if err != nil {
			return nil, fmt.Errorf("cannot read orcid response: %s", err)
		}
		if err = xml.Unmarshal(b, data); err != nil {
			return nil, fmt.Errorf("cannot decode orcid response: %s [response: %s]", err, b)
		}
	}
	return res, err
}

func IsID(id string) bool {
	return reID.MatchString(id)
}
