package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sort"
	"time"

	ragkit "github.com/suapapa/go_ragkit"
	ragkit_helper "github.com/suapapa/go_ragkit/helper"
)

var vstorePhrases ragkit.VectorStore

func initVStorePhrases() error {
	if vstorePhrases != nil {
		return nil
	}

	tablePhrases := envOr("PGVECTOR_TABLE_PHRASES", "homin_dev_phrases_openai")
	embedModel := envOr("OPENAI_EMBED_MODEL", ragkit_helper.DefaultOAIEmbedModel())

	var err error
	vstorePhrases, err = ragkit_helper.NewPGVectorOpenAIVectorStore(
		tablePhrases,
		embedModel,
	)
	if err != nil {
		return fmt.Errorf("failed to create vector store: %w", err)
	}
	return nil
}

func retrivePost(prompt string, cnt int) ([]*Post, error) {
	if err := initVStorePhrases(); err != nil {
		return nil, err
	}

	log.Println("retrieving post with prompt:", prompt)
	docs, err := vstorePhrases.RetrieveText(context.Background(), prompt, cnt, "title", "post_url", "tags", "date")
	if err != nil {
		return nil, fmt.Errorf("failed to search phrases: %w", err)
	}

	postMap := make(map[string]*Post)

	for _, doc := range docs {
		postUrl := doc.Metadata["post_url"].(string)
		title := doc.Metadata["title"].(string)
		dateStr := doc.Metadata["date"].(string)
		date, err := time.Parse(time.RFC3339, dateStr)
		if err != nil {
			log.Printf("failed to parse date: %v", err)
		}
		var tags []string
		if tagInterfaces, ok := doc.Metadata["tags"].([]interface{}); ok {
			for _, tagInterface := range tagInterfaces {
				if tagStr, ok := tagInterface.(string); ok {
					tags = append(tags, tagStr)
				}
			}
		}
		text := doc.Text

		if _, ok := postMap[postUrl]; !ok {
			postMap[postUrl] = &Post{
				Title: title,
				Url:   postUrl,
				Tags:  tags,
				Texts: []string{text},
				Date:  date,
			}
		} else {
			postMap[postUrl].Texts = append(postMap[postUrl].Texts, text)
		}
	}

	posts := make(Posts, 0, len(postMap))
	for _, post := range postMap {
		posts = append(posts, post)
	}
	sort.Sort(posts)

	return posts, nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

type Post struct {
	Title string    `yaml:"title"`
	Url   string    `yaml:"url"`
	Tags  []string  `yaml:"tags"`
	Texts []string  `yaml:"texts"`
	Date  time.Time `yaml:"date"`
}

type Posts []*Post

func (p Posts) Len() int {
	return len(p)
}

func (p Posts) Less(i, j int) bool {
	if len(p[i].Texts) == len(p[j].Texts) {
		return p[i].Date.After(p[j].Date)
	}
	return len(p[i].Texts) > len(p[j].Texts)
}

func (p Posts) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}
