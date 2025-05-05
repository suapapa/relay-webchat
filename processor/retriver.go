package main

import (
	"context"
	"fmt"
	"log"

	ragkit "github.com/suapapa/go_ragkit"
	ragkit_helper "github.com/suapapa/go_ragkit/helper"
)

var vstorePhrases ragkit.VectorStore

func init() {
	var err error
	vstorePhrases, err = ragkit_helper.NewWeaviateOllamaVectorStore(
		"homin_dev_phrases_ollama", // vector DB class name
		ragkit_helper.DefaultOllamaEmbedModel,
	)
	if err != nil {
		log.Fatalf("failed to create vector store: %v", err)
	}
}

func retrivePost(prompt string, cnt int) ([]*Post, error) {
	log.Println("retrieving post with prompt:", prompt)
	docs, err := vstorePhrases.RetrieveText(context.Background(), prompt, cnt, "title", "post_url", "tags")
	if err != nil {
		return nil, fmt.Errorf("failed to search phrases: %w", err)
	}

	posts := make([]*Post, 0, len(docs))
	for _, doc := range docs {
		var tags []string

		if tags, ok := doc.Metadata["tags"].([]any); ok {
			for _, tag := range doc.Metadata["tags"].([]any) {
				tags = append(tags, tag.(string))
			}
		}

		posts = append(posts, &Post{
			Title: doc.Metadata["title"].(string),
			Url:   doc.Metadata["post_url"].(string),
			Tags:  tags,
		})
	}

	return posts, nil
}

type Post struct {
	Title string   `yaml:"title"`
	Url   string   `yaml:"url"`
	Tags  []string `yaml:"tags"`
}
