package opensearchswitcher

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"regexp"
	"sort"
	"time"

	"github.com/opensearch-project/opensearch-go/v4/opensearchapi"
	"github.com/opensearch-project/opensearch-go/v4/opensearchutil"
)

const timeFormat = "20060102150405"

type Switcher[T any] struct {
	client      *opensearchapi.Client
	bulkIndexer opensearchutil.BulkIndexer
	alias       string
	retention   int
	ItemFunc    func(T) (opensearchutil.BulkIndexerItem, error)
}

type Config[T any] struct {
	Client        *opensearchapi.Client
	Alias         string
	IndexSettings io.ReadSeeker
	Retention     int
	ItemFunc      func(T) (opensearchutil.BulkIndexerItem, error)
}

func Init(ctx context.Context, client *opensearchapi.Client, alias string, indexSettings io.ReadSeeker, retention int) error {
	exists, err := aliasExists(ctx, client, alias)
	if err != nil {
		return err
	}
	if !exists {
		if _, err := createIndex(ctx, client, alias, indexSettings); err != nil {
			return err
		}

		if err := switchAlias(ctx, client, alias, retention); err != nil {
			return err
		}
	}

	return nil
}

func New[T any](ctx context.Context, c Config[T]) (*Switcher[T], error) {
	idx := fmt.Sprintf("%s_%s", c.Alias, time.Now().UTC().Format(timeFormat))

	_, err := c.Client.Indices.Create(ctx, opensearchapi.IndicesCreateReq{
		Index: idx,
		Body:  c.IndexSettings,
	})
	if err != nil {
		return nil, err
	}

	bulkIndexer, err := opensearchutil.NewBulkIndexer(opensearchutil.BulkIndexerConfig{
		Client: c.Client,
		Index:  idx,
		// TODO make configurable
		OnError: func(_ context.Context, err error) {
			log.Printf("error indexing: %s", err)
		},
	})
	if err != nil {
		return nil, err
	}

	return &Switcher[T]{
		client:      c.Client,
		alias:       c.Alias,
		retention:   c.Retention,
		bulkIndexer: bulkIndexer,
		ItemFunc:    c.ItemFunc,
	}, nil
}

// TODO error handling (but don't error on version conflict or already exists error)
func (switcher Switcher[T]) Add(ctx context.Context, t T) error {
	item, err := switcher.ItemFunc(t)
	if err != nil {
		return err
	}

	item.Action = "index"
	// TODO make configurable
	item.OnFailure = func(_ context.Context, bii opensearchutil.BulkIndexerItem, _ opensearchapi.BulkRespItem, err error) {
		log.Printf("error indexing %s: %s", bii.DocumentID, err)
	}

	return switcher.bulkIndexer.Add(ctx, item)
}

func (switcher Switcher[T]) Switch(ctx context.Context) error {
	if err := switcher.bulkIndexer.Close(ctx); err != nil {
		return err
	}
	return switchAlias(ctx, switcher.client, switcher.alias, switcher.retention)
}

func createIndex(ctx context.Context, client *opensearchapi.Client, alias string, indexSettings io.ReadSeeker) (string, error) {
	idx := fmt.Sprintf("%s_%s", alias, time.Now().UTC().Format(timeFormat))

	_, err := client.Indices.Create(ctx, opensearchapi.IndicesCreateReq{
		Index: idx,
		Body:  indexSettings,
	})
	if err != nil {
		return "", err
	}
	return idx, nil
}

type aliasAction struct {
	Alias string `json:"alias,omitempty"`
	Index string `json:"index,omitempty"`
}

func switchAlias(ctx context.Context, client *opensearchapi.Client, alias string, retention int) error {
	indices, err := existingIndices(ctx, client, alias)
	if err != nil {
		return err
	}

	var actions []map[string]aliasAction

	for i, idx := range indices {
		var a map[string]aliasAction
		switch {
		case i == 0:
			a = map[string]aliasAction{"add": {Alias: alias, Index: idx}}
		case i <= retention:
			a = map[string]aliasAction{"remove": {Alias: alias, Index: idx}}
		default:
			a = map[string]aliasAction{"remove_index": {Index: idx}}
		}
		actions = append(actions, a)
	}

	body, err := json.Marshal(map[string][]map[string]aliasAction{"actions": actions})
	if err != nil {
		return err
	}

	_, err = client.Aliases(ctx, opensearchapi.AliasesReq{
		Body: bytes.NewReader(body),
	})

	return err
}

// sorted new to old, first one is the new index we are switching to
func existingIndices(ctx context.Context, client *opensearchapi.Client, alias string) ([]string, error) {
	res, err := client.Cat.Indices(ctx, &opensearchapi.CatIndicesReq{
		Indices: []string{alias + "_*"},
	})
	if err != nil {
		return nil, err
	}

	re := regexp.MustCompile(fmt.Sprintf(`^%s_[0-9]{%d}$`, alias, len(timeFormat)))

	var indices []string
	for _, idx := range res.Indices {
		// in case our wildcard was too broad
		if re.MatchString(idx.Index) {
			indices = append(indices, idx.Index)
		}
	}

	sort.Sort(sort.Reverse(sort.StringSlice(indices)))

	return indices, nil
}

func aliasExists(ctx context.Context, client *opensearchapi.Client, alias string) (bool, error) {
	res, err := client.Indices.Alias.Exists(ctx, opensearchapi.AliasExistsReq{
		Alias: []string{alias},
	})
	if res != nil && res.StatusCode == 200 {
		return true, nil
	}
	if res != nil && res.StatusCode == 404 {
		return false, nil
	}
	return false, err
}
