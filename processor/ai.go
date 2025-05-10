package main

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/core"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/googlegenai"
)

type HominDevAI struct {
	PreProcessFLow *core.Flow[string, Cmd, struct{}]
	// SearchPostFlow *core.Flow[string, Result, struct{}]

	mu sync.Mutex
}

func NewHominDevAI(ctx context.Context) (*HominDevAI, error) {
	// o := &ollama_plugin.Ollama{
	// 	ServerAddress: cmp.Or(os.Getenv("OLLAMA_ADDR"), "http://localhost:11434"),
	// }
	// g, err := genkit.Init(
	// 	ctx,
	// 	genkit.WithPlugins(o),
	// 	genkit.WithDefaultModel("ollama/qwen3"),
	// )
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to initialize genkit: %w", err)
	// }

	// o.DefineModel(
	// 	g,
	// 	ollama_plugin.ModelDefinition{
	// 		Name: "qwen3",
	// 		Type: "chat",
	// 	},
	// 	&ai.ModelInfo{
	// 		Label: "QWEN 3",
	// 		Supports: &ai.ModelSupports{
	// 			Multiturn:  true,
	// 			SystemRole: true,
	// 			Media:      true,
	// 			Tools:      false,
	// 		},
	// 	},
	// )

	g, err := genkit.Init(ctx,
		genkit.WithPlugins(
			&googlegenai.GoogleAI{},
		),
		genkit.WithDefaultModel("googleai/gemini-2.0-flash"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Genkit: %w", err)
	}

	ret := &HominDevAI{}

	ret.PreProcessFLow = genkit.DefineFlow(
		g, "preProcessFlow",
		func(ctx context.Context, input string) (Cmd, error) {
			ret.mu.Lock()
			defer ret.mu.Unlock()

			if flagPromptPreProcess {
				s, _, err := genkit.GenerateData[Cmd](
					ctx, g,
					ai.WithSystem(preProcessSystemPrompt),
					ai.WithPrompt(fmt.Sprintf(preProcessUserPromptFmt, input)),
				)
				if err != nil {
					log.Printf("failed to find keywords: %s", err)
					return Cmd{"/about", []string{}}, nil
				}
				return *s, nil
			} else {
				return Cmd{"/search", []string{}}, nil
			}
		},
	)

	/*
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
	*/

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
	preProcessSystemPrompt  = `너의 이름은 블검봇. 사용자와 간단한 잡담을 하거나 사용자가 궁금해 하는 키워드를 추출하는 봇이야.
사용자가 블로그에 대해 궁금한 것 같으면 사용자의 입력에서 사용자가 궁금해하는 키워드를, keyword 을 추출해.
상황에 따라서 키워드를 추출할 수 없으면 간단한 대꾸, smallchat 을 출력해

- 사용자의 의도에 따라 **반드시 아래 중 하나의 행동을 출력**해야 해.
- 행동(Action) 목록:
  - /keyword: 사용자가 궁금해 하는 키워드를 추출해야 할 때
  - /smallchat: 사용자와 소소한 대화를 할 때
  - /about: 봇의 정보를 출력해야 할 때

출력 규칙:
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
