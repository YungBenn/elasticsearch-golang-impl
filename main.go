package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/labstack/echo/v4"
)

type Product struct {
	Name        string `json:"name"`
	Price       int    `json:"price"`
	Tag         string `json:"tag"`
	Description string `json:"description"`
}

func NewElasticsearch() (*elasticsearch.Client, error) {
	es, err := elasticsearch.NewDefaultClient()
	if err != nil {
		panic(err)
	}

	return es, err
}

func CreateIndex(ctx context.Context, index string) error {
	es, err := NewElasticsearch()
	if err != nil {
		return fmt.Errorf("error creating new Elasticsearch: %w", err)
	}

	mapping := `{
		"settings": {
			"number_of_shards": 1
			},
			"mappings": {
			"properties": {
				"field1": {
				"type": "text"
				}
			}
		}
	}`

	_, err = es.Indices.Create(
		index,
		es.Indices.Create.WithBody(strings.NewReader(mapping)),
	)
	if err != nil {
		return fmt.Errorf("error creating index: %w", err)
	}

	return nil
}

func Index(ctx context.Context) error {
	es, err := NewElasticsearch()
	if err != nil {
		return errors.New("error when creating new elasticsearch client")
	}

	body := Product{
		Name:        "Iphone 14 Pro",
		Price:       1000000,
		Tag:         "Smartphone",
		Description: "Iphone 14 Pro with 1TB storage",
	}

	data, err := json.Marshal(body)
	if err != nil {
		return errors.New("error when marshalling data")
	}

	_, err = es.Index("hello", bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("error indexing data: %w", err)
	}

	return nil
}

func Search(ctx context.Context, query string) ([]Product, error) {
	es, err := NewElasticsearch()
	if err != nil {
		return nil, errors.New("error when creating new elasticsearch client")
	}

	req := `{
		"query": {
			"multi_match": {
				"query": "` + query + `",
				"type": "phrase_prefix",
				"fields": ["name", "tag", "description"]
			}
		}
	}`

	res, err := es.Search(
		es.Search.WithContext(ctx),
		es.Search.WithIndex("hello"),
		es.Search.WithBody(strings.NewReader(req)),
		es.Search.WithTrackTotalHits(true),
		es.Search.WithPretty(),
	)
	if err != nil {
		return nil, errors.New("error when searching data")
	}

	defer res.Body.Close()

	var result map[string]interface{}

	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, errors.New("error when decoding data")
	}

	var products []Product

	for _, hit := range result["hits"].(map[string]interface{})["hits"].([]interface{}) {
		product := Product{}
		data, _ := json.Marshal(hit.(map[string]interface{})["_source"])
		json.Unmarshal(data, &product)

		products = append(products, product)
	}

	return products, nil
}

func main() {
	e := echo.New()

	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "Hello, World!1")
	})

	e.POST("/create", func(c echo.Context) error {
		err := CreateIndex(context.TODO(), c.QueryParam("index"))
		if err != nil {
			return c.String(http.StatusInternalServerError, err.Error())
		}

		return c.String(http.StatusOK, "Hello, World!2")
	})

	e.POST("/index", func(c echo.Context) error {
		err := Index(context.Background())
		if err != nil {
			return c.String(http.StatusInternalServerError, err.Error())
		}

		return c.String(http.StatusOK, "Hello, World!3")
	})

	e.GET("/search", func(c echo.Context) error {
		res, err := Search(context.Background(), c.QueryParam("search"))
		if err != nil {
			return c.String(http.StatusInternalServerError, "Error when indexing data")
		}

		fmt.Println(res)

		return c.JSON(http.StatusOK, res)
	})

	e.Logger.Fatal(e.Start(":4000"))
}
