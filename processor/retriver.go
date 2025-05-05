package main

import (
	"context"
	"fmt"
	"log"
	"sort"

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

	postMap := make(map[string]*Post)

	for _, doc := range docs {
		postUrl := doc.Metadata["post_url"].(string)
		title := doc.Metadata["title"].(string)
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

type Post struct {
	Title string   `yaml:"title"`
	Url   string   `yaml:"url"`
	Tags  []string `yaml:"tags"`
	Texts []string `yaml:"texts"`
}

type Posts []*Post

func (p Posts) Len() int {
	return len(p)
}

func (p Posts) Less(i, j int) bool {
	return len(p[i].Texts) > len(p[j].Texts)
}

func (p Posts) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}
