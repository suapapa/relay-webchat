package main

import (
	"cmp"
	"context"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/core"
	"github.com/firebase/genkit/go/genkit"
	ollama_plugin "github.com/firebase/genkit/go/plugins/ollama"
	"github.com/goccy/go-yaml"
)

type HominDevAI struct {
	IntentFLow     *core.Flow[string, Cmd, struct{}]
	SearchPostFlow *core.Flow[string, Result, struct{}]

	mu sync.Mutex
}

func NewHominDevAI(ctx context.Context) (*HominDevAI, error) {
	o := &ollama_plugin.Ollama{
		ServerAddress: cmp.Or(os.Getenv("OLLAMA_ADDR"), "http://localhost:11434"),
	}
	g, err := genkit.Init(
		ctx,
		genkit.WithPlugins(o),
		genkit.WithDefaultModel("ollama/gemma3"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize genkit: %w", err)
	}

	o.DefineModel(
		g,
		ollama_plugin.ModelDefinition{
			Name: "gemma3",
			Type: "chat",
		},
		&ai.ModelInfo{
			Label: "Gemma 3",
			Supports: &ai.ModelSupports{
				Multiturn:  true,
				SystemRole: true,
				Media:      true,
				Tools:      false,
			},
		},
	)
	ret := &HominDevAI{}

	intentFlow := genkit.DefineFlow(
		g, "intentFlow",
		func(ctx context.Context, input string) (Cmd, error) {
			ret.mu.Lock()
			defer ret.mu.Unlock()

			s, _, err := genkit.GenerateData[Cmd](
				ctx, g,
				ai.WithSystem(intentSystemPrompt),
				ai.WithPrompt(fmt.Sprintf(intentUserPromptFmt, input)),
			)
			if err != nil {
				log.Printf("failed to judge intent: %s", err)
				return Cmd{"/search", []string{}}, nil
			}
			return *s, nil
		},
	)
	ret.IntentFLow = intentFlow

	searchPostFlow := genkit.DefineFlow(
		g, "searchPostFlow",
		func(ctx context.Context, input string) (Result, error) {
			ret.mu.Lock()
			defer ret.mu.Unlock()

			log.Printf("searchPostFlow: %s", input)

			postCandidates, err := retrivePost(input, flagRetriveCnt)
			if err != nil {
				return Result{}, fmt.Errorf("failed to retrieve post: %w", err)
			}

			if len(postCandidates) == 0 {
				log.Printf("no post candidates found")
				return Result{}, nil
			}

			genkitDocs := make([]*ai.Document, 0, len(postCandidates))
			for _, post := range postCandidates {
				yamlPost, err := yaml.Marshal(post)
				if err != nil {
					log.Printf("failed to marshal post: %s", err)
					continue
				}
				genkitDocs = append(genkitDocs, &ai.Document{
					Content: []*ai.Part{ai.NewTextPart(string(yamlPost))},
				})
			}

			r, _, err := genkit.GenerateData[Result](
				ctx, g,
				ai.WithDocs(genkitDocs...),
				ai.WithSystem(searchSystemPrompt),
				ai.WithPrompt(fmt.Sprintf(searchUserPromptFmt, input)),
			)
			if err != nil {
				return Result{}, fmt.Errorf("failed to generate search result: %w", err)
			}

			return *r, nil
		},
	)
	ret.SearchPostFlow = searchPostFlow

	return ret, nil
}

type Result struct {
	Posts []*Post `json:"posts" yaml:"posts"`
}

type Cmd struct {
	Action string   `json:"action" yaml:"action"` // search, smallchat
	Args   []string `json:"args" yaml:"args"`
}

var (
	searchUserPromptFmt = "사용자 입력: %s"
	searchSystemPrompt  = `너는 블로그 포스팅을 검색하는 사서 봇이야.
제공된 포스팅 목록은, 벡터디비에서 사용자의 입력에 대해 유사도를 검색한 결과로 포스팅의 제목들이야.
포스팅의 제목들을 살펴보고 사용자 입력과 연관이 높은 순으로 출력.

입출력 아이템 키 의미:
- title: 포스팅 제목
- url: 포스팅 링크

출력규칙:
- 전달받은 오브젝트의 posts 필드에 결과 포스팅들을 출력.
- 포스팅의 태그가 사용자 입력과 연관이 있으면 출력.
- 포스팅 링크가 같은 포스팅은 출력하지 않는다.
- 최대 10개의 포스팅 링크를 출력해.
`

	intentUserPromptFmt = "다음은 사용자의 채팅 메시지:\n%s"
	intentSystemPrompt  = `너의 임무는 사용자의 채팅 메시지를 분석하여 적절한 행동(Action)을 선택하는 것이야.

- 사용자의 의도에 따라 **반드시 아래 중 하나의 행동을 출력**해야 해.
- 행동(Action) 목록:
  - /search: 검색을 해야 할 때
  - /smallchat: 사용자와 소소한 대화를 할 때
  - /about: 봇의 정보를 출력해야 할 때

출력 규칙:
- action 필드에는 오직 행동 이름(/search, /smallchat, /about)만 출력한다.
- /smallchat 행동에 대해서는 네가 검색봇이라는 내용으로 적적한 대꾸를 각 줄을 args의 각 요소로 입력한다.
- args 필드에는 추가적인 정보가 없다면 빈 배열을 출력한다.
- 다른 문장이나 설명은 절대 추가하지 않는다.
- 여러 행동이 떠오르거나 아무 생각이 떠오르지 않으면 /search 행동을 선택한다.

주의사항:
- 행동 이름은 반드시 / 로 시작하는 소문자 단어로 출력한다.
`
)
