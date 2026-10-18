package main

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	oai "github.com/openai/openai-go"
	oai_option "github.com/openai/openai-go/option"
	"github.com/openai/openai-go/shared"
)

const defaultChatModel = "gpt-4o-mini"

type HominDevAI struct {
	client *oai.Client
	model  string

	mu sync.Mutex
}

func NewHominDevAI(_ context.Context) (*HominDevAI, error) {
	opts := []oai_option.RequestOption{
		oai_option.WithAPIKey(os.Getenv("OPENAI_APIKEY")),
	}
	if ep := os.Getenv("OPENAI_ENDPOINT"); ep != "" {
		opts = append(opts, oai_option.WithBaseURL(ep))
	}

	client := oai.NewClient(opts...)
	model := cmp.Or(os.Getenv("OPENAI_CHAT_MODEL"), defaultChatModel)

	return &HominDevAI{
		client: &client,
		model:  model,
	}, nil
}

func (a *HominDevAI) PreProcess(ctx context.Context, input string) (Cmd, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !flagPromptPreProcess {
		return Cmd{Action: "/search", Args: []string{}}, nil
	}

	resp, err := a.client.Chat.Completions.New(ctx, oai.ChatCompletionNewParams{
		Model: a.model,
		Messages: []oai.ChatCompletionMessageParamUnion{
			oai.SystemMessage(preProcessSystemPrompt),
			oai.UserMessage(fmt.Sprintf(preProcessUserPromptFmt, input)),
		},
		ResponseFormat: oai.ChatCompletionNewParamsResponseFormatUnion{
			OfJSONSchema: &shared.ResponseFormatJSONSchemaParam{
				JSONSchema: shared.ResponseFormatJSONSchemaJSONSchemaParam{
					Name:   "cmd",
					Strict: oai.Bool(true),
					Schema: cmdJSONSchema,
				},
			},
		},
	})
	if err != nil {
		log.Printf("failed to find keywords: %v", err)
		return Cmd{Action: "/about", Args: []string{}}, nil
	}
	if len(resp.Choices) == 0 {
		log.Printf("failed to find keywords: empty choices")
		return Cmd{Action: "/about", Args: []string{}}, nil
	}

	raw := extractJSON(resp.Choices[0].Message.Content)
	var cmd Cmd
	if err := json.Unmarshal([]byte(raw), &cmd); err != nil {
		log.Printf("failed to parse command JSON (%s): %v", raw, err)
		return Cmd{Action: "/about", Args: []string{}}, nil
	}
	return cmd, nil
}

func extractJSON(s string) string {
	if idx := strings.Index(s, "</think>"); idx != -1 {
		s = s[idx+len("</think>"):]
	}
	start := strings.IndexByte(s, '{')
	end := strings.LastIndexByte(s, '}')
	if start != -1 && end != -1 && end >= start {
		return s[start : end+1]
	}
	return strings.TrimSpace(s)
}

type Result struct {
	Posts []*Post `json:"posts" yaml:"posts"`
}

type Cmd struct {
	Action string   `json:"action" yaml:"action"` // search, smallchat
	Args   []string `json:"args" yaml:"args"`
}

var cmdJSONSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"action": map[string]any{
			"type": "string",
			"enum": []string{"/keyword", "/smallchat", "/about"},
		},
		"args": map[string]any{
			"type": "array",
			"items": map[string]any{
				"type": "string",
			},
		},
	},
	"required":             []string{"action", "args"},
	"additionalProperties": false,
}

var (
	// 	searchUserPromptFmt = "사용자 입력: %s"
	// 	searchSystemPrompt  = `너는 블로그 포스팅을 검색하는 사서 봇이야.
	// 제공된 포스팅 목록은, 벡터디비에서 사용자의 입력에 대해 유사도를 검색한 결과로 포스팅의 제목들이야.
	// 포스팅의 제목들을 살펴보고 사용자 입력과 연관이 높은 순으로 출력.

	// 입출력 아이템 키 의미:
	// - title: 포스팅 제목
	// - url: 포스팅 링크

	// 출력규칙:
	// - 전달받은 오브젝트의 posts 필드에 결과 포스팅들을 출력.
	// - 포스팅의 태그가 사용자 입력과 연관이 있으면 출력.
	// - 포스팅 링크가 같은 포스팅은 출력하지 않는다.
	// - 최대 10개의 포스팅 링크를 출력해.
	// `

	preProcessUserPromptFmt = "다음은 사용자의 입력:\n%s"
	preProcessSystemPrompt  = `너의 이름은 블검봇. 사용자와 간단한 잡담을 하거나
사용자가 궁금해 하는 블로그 검색 키워드를 추출하는 봇이야.
사용자가 블로그에 대해 궁금한 것 같으면 사용자의 입력에서 사용자가 궁금해하는 키워드, keyword 을 추출해.
상황에 따라서 키워드를 추출할 수 없으면 간단한 대꾸, smallchat 을 출력해.
smallchat 을 할 때는 사용자가 너를 블로그 검색에 사용하도록 유도해 줘.

- 사용자의 의도에 따라 **반드시 아래 중 하나의 행동을 출력**해야 해.
- 행동(Action) 목록:
  - /keyword: 사용자가 궁금해 하는 키워드를 추출해야 할 때
  - /smallchat: 사용자와 소소한 대화를 할 때
  - /about: 봇의 정보를 출력해야 할 때

출력 규칙:
- 반드시 {"action": "/keyword", "args": ["..."]} 형태의 JSON 객체로만 출력한다.
- action 필드에는 오직 행동 이름(/keyword, /smallchat, /about)만 출력한다.
- /smallchat 행동에 대해서는 적절한 대꾸를 각 줄을 args의 각 요소로 입력한다.
- /keyword 행동에 대해서는 사용자가 궁금해 하는 키워드를을 args의 각 요소로 입력한다.
- args 필드에는 추가적인 정보가 없다면 빈 배열을 출력한다.
- 다른 문장이나 설명은 절대 추가하지 않는다.
- 여러 행동이 떠오르거나 아무 생각이 떠오르지 않으면 /smallchat 행동을 선택한다.

주의사항:
- 행동 이름은 반드시 / 로 시작하는 소문자 단어로 출력한다.
`
)

