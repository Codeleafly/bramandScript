package vm

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"sort"
	"strings"
	"time"

	"bramandscript/object"
)

func init() {
	RegisterBuiltins(map[string]*object.Builtin{
		"ai_prompt": {
			Fn: func(args ...object.Object) object.Object {
				if len(args) < 2 {
					return &object.Error{Message: "ai_prompt requires at least 2 arguments (model, prompt)"}
				}
				model := args[0].Inspect()
				if sObj, ok := args[0].(*object.String); ok {
					model = sObj.Value
				}
				prompt := args[1].Inspect()
				if sObj, ok := args[1].(*object.String); ok {
					prompt = sObj.Value
				}

				endpoint := "http://localhost:11434/api/generate"
				apiKey := ""
				var systemPrompt string
				if len(args) >= 3 {
					if opts, ok := args[2].(*object.Hash); ok {
						for _, pair := range opts.Pairs {
							k := pair.Key.Inspect()
							if k == "endpoint" || k == "url" {
								endpoint = pair.Value.Inspect()
								if s, ok := pair.Value.(*object.String); ok {
									endpoint = s.Value
								}
							} else if k == "api_key" {
								apiKey = pair.Value.Inspect()
								if s, ok := pair.Value.(*object.String); ok {
									apiKey = s.Value
								}
							} else if k == "system" {
								systemPrompt = pair.Value.Inspect()
								if s, ok := pair.Value.(*object.String); ok {
									systemPrompt = s.Value
								}
							}
						}
					}
				}

				isOpenAI := strings.Contains(endpoint, "/v1/") || strings.Contains(endpoint, "chat/completions")
				var reqBody []byte
				if isOpenAI {
					messages := []map[string]string{}
					if systemPrompt != "" {
						messages = append(messages, map[string]string{"role": "system", "content": systemPrompt})
					}
					messages = append(messages, map[string]string{"role": "user", "content": prompt})
					payload := map[string]interface{}{
						"model":    model,
						"messages": messages,
						"stream":   false,
					}
					reqBody, _ = json.Marshal(payload)
				} else {
					payload := map[string]interface{}{
						"model":  model,
						"prompt": prompt,
						"stream": false,
					}
					if systemPrompt != "" {
						payload["system"] = systemPrompt
					}
					reqBody, _ = json.Marshal(payload)
				}

				req, err := http.NewRequest("POST", endpoint, bytes.NewReader(reqBody))
				if err != nil {
					return &object.Error{Message: fmt.Sprintf("ai_prompt request creation failed: %s", err)}
				}
				req.Header.Set("Content-Type", "application/json")
				if apiKey != "" {
					req.Header.Set("Authorization", "Bearer "+apiKey)
				}

				client := &http.Client{Timeout: 60 * time.Second}
				resp, err := client.Do(req)
				if err != nil {
					return &object.Error{Message: fmt.Sprintf("ai_prompt request failed: %s", err)}
				}
				defer resp.Body.Close()

				bodyBytes, err := io.ReadAll(resp.Body)
				if err != nil {
					return &object.Error{Message: fmt.Sprintf("reading ai_prompt response failed: %s", err)}
				}

				var result map[string]interface{}
				if err := json.Unmarshal(bodyBytes, &result); err != nil {
					return &object.String{Value: string(bodyBytes)}
				}

				if respText, ok := result["response"].(string); ok {
					return &object.String{Value: respText}
				}
				if choices, ok := result["choices"].([]interface{}); ok && len(choices) > 0 {
					if c0, ok := choices[0].(map[string]interface{}); ok {
						if msg, ok := c0["message"].(map[string]interface{}); ok {
							if content, ok := msg["content"].(string); ok {
								return &object.String{Value: content}
							}
						}
					}
				}
				return &object.String{Value: string(bodyBytes)}
			},
		},
		"ai_stream": {
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 2 {
					return &object.Error{Message: "ai_stream requires 2 arguments (options, chunk_callback)"}
				}
				opts, ok := args[0].(*object.Hash)
				if !ok {
					return &object.Error{Message: "first argument to ai_stream must be options Hash"}
				}
				callback := args[1]

				endpoint := "http://localhost:11434/api/generate"
				var reqBody []byte
				apiKey := ""
				for _, pair := range opts.Pairs {
					k := pair.Key.Inspect()
					if k == "url" || k == "endpoint" {
						endpoint = pair.Value.Inspect()
						if s, ok := pair.Value.(*object.String); ok {
							endpoint = s.Value
						}
					} else if k == "api_key" {
						apiKey = pair.Value.Inspect()
						if s, ok := pair.Value.(*object.String); ok {
							apiKey = s.Value
						}
					} else if k == "body" {
						if s, ok := pair.Value.(*object.String); ok {
							reqBody = []byte(s.Value)
						} else {
							reqBody, _ = json.Marshal(bramandObjectToNative(pair.Value))
						}
					}
				}

				req, err := http.NewRequest("POST", endpoint, bytes.NewReader(reqBody))
				if err != nil {
					return &object.Error{Message: fmt.Sprintf("ai_stream request creation failed: %s", err)}
				}
				req.Header.Set("Content-Type", "application/json")
				if apiKey != "" {
					req.Header.Set("Authorization", "Bearer "+apiKey)
				}

				client := &http.Client{Timeout: 120 * time.Second}
				resp, err := client.Do(req)
				if err != nil {
					return &object.Error{Message: fmt.Sprintf("ai_stream connection failed: %s", err)}
				}
				defer resp.Body.Close()

				scanner := bufio.NewScanner(resp.Body)
				var accumulated strings.Builder
				for scanner.Scan() {
					line := strings.TrimSpace(scanner.Text())
					if line == "" || line == "data: [DONE]" {
						continue
					}
					cleanJSON := line
					if strings.HasPrefix(line, "data: ") {
						cleanJSON = strings.TrimPrefix(line, "data: ")
					}

					var chunkMap map[string]interface{}
					if err := json.Unmarshal([]byte(cleanJSON), &chunkMap); err != nil {
						_, _ = ExecuteCallable(callback, &object.String{Value: cleanJSON})
						accumulated.WriteString(cleanJSON)
						continue
					}

					var chunkText string
					if r, ok := chunkMap["response"].(string); ok {
						chunkText = r
					} else if choices, ok := chunkMap["choices"].([]interface{}); ok && len(choices) > 0 {
						if c0, ok := choices[0].(map[string]interface{}); ok {
							if delta, ok := c0["delta"].(map[string]interface{}); ok {
								if c, ok := delta["content"].(string); ok {
									chunkText = c
								}
							}
						}
					}

					if len(chunkText) > 0 {
						accumulated.WriteString(chunkText)
						_, _ = ExecuteCallable(callback, &object.String{Value: chunkText})
					}
				}

				return &object.String{Value: accumulated.String()}
			},
		},
		"ai_structured": {
			Fn: func(args ...object.Object) object.Object {
				if len(args) < 2 {
					return &object.Error{Message: "ai_structured requires 2 arguments (options, schema_or_example)"}
				}
				schemaJSON, _ := json.Marshal(bramandObjectToNative(args[1]))
				schemaInstruction := fmt.Sprintf("\nYou MUST respond ONLY with valid, raw JSON matching this format: %s\nDo not include any conversational preamble or markdown code fences.", string(schemaJSON))

				var model, prompt, endpoint, apiKey string
				endpoint = "http://localhost:11434/api/generate"
				if opts, ok := args[0].(*object.Hash); ok {
					for _, pair := range opts.Pairs {
						k := pair.Key.Inspect()
						if k == "model" {
							model = pair.Value.Inspect()
							if s, ok := pair.Value.(*object.String); ok {
								model = s.Value
							}
						} else if k == "prompt" {
							prompt = pair.Value.Inspect()
							if s, ok := pair.Value.(*object.String); ok {
								prompt = s.Value
							}
						} else if k == "endpoint" || k == "url" {
							endpoint = pair.Value.Inspect()
							if s, ok := pair.Value.(*object.String); ok {
								endpoint = s.Value
							}
						} else if k == "api_key" {
							apiKey = pair.Value.Inspect()
							if s, ok := pair.Value.(*object.String); ok {
								apiKey = s.Value
							}
						}
					}
				}

				fullPrompt := prompt + schemaInstruction
				promptBuiltin := getBuiltinByName("ai_prompt")
				optsPairs := make(map[object.HashKey]object.HashPair)
				setOpt := func(k, v string) {
					kObj := &object.String{Value: k}
					optsPairs[kObj.HashKey()] = object.HashPair{Key: kObj, Value: &object.String{Value: v}}
				}
				setOpt("endpoint", endpoint)
				if apiKey != "" {
					setOpt("api_key", apiKey)
				}

				rawResp := promptBuiltin.Fn(&object.String{Value: model}, &object.String{Value: fullPrompt}, &object.Hash{Pairs: optsPairs})
				if errObj, ok := rawResp.(*object.Error); ok {
					return errObj
				}

				respStr := rawResp.Inspect()
				if sObj, ok := rawResp.(*object.String); ok {
					respStr = sObj.Value
				}

				cleaned := strings.TrimSpace(respStr)
				if strings.HasPrefix(cleaned, "```json") {
					cleaned = strings.TrimPrefix(cleaned, "```json")
					cleaned = strings.TrimSuffix(cleaned, "```")
				} else if strings.HasPrefix(cleaned, "```") {
					cleaned = strings.TrimPrefix(cleaned, "```")
					cleaned = strings.TrimSuffix(cleaned, "```")
				}
				cleaned = strings.TrimSpace(cleaned)

				var parsed interface{}
				if err := json.Unmarshal([]byte(cleaned), &parsed); err != nil {
					return &object.Error{Message: fmt.Sprintf("failed to parse structured JSON: %s. Response was: %q", err, cleaned)}
				}
				return nativeToBramandObject(parsed)
			},
		},
		"cosine_similarity": {
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 2 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=2", len(args))}
				}
				v1, err1 := extractVectorSlice(args[0])
				v2, err2 := extractVectorSlice(args[1])
				if err1 != nil || err2 != nil {
					return &object.Error{Message: "arguments to cosine_similarity must be vectors or arrays of numbers"}
				}
				sim, err := calcCosineSimilarity(v1, v2)
				if err != nil {
					return &object.Error{Message: err.Error()}
				}
				return &object.Float{Value: sim}
			},
		},
		"cosine_distance": {
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 2 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=2", len(args))}
				}
				v1, err1 := extractVectorSlice(args[0])
				v2, err2 := extractVectorSlice(args[1])
				if err1 != nil || err2 != nil {
					return &object.Error{Message: "arguments to cosine_distance must be vectors or arrays of numbers"}
				}
				sim, err := calcCosineSimilarity(v1, v2)
				if err != nil {
					return &object.Error{Message: err.Error()}
				}
				return &object.Float{Value: 1.0 - sim}
			},
		},
		"euclidean_distance": {
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 2 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=2", len(args))}
				}
				v1, err1 := extractVectorSlice(args[0])
				v2, err2 := extractVectorSlice(args[1])
				if err1 != nil || err2 != nil {
					return &object.Error{Message: "arguments to euclidean_distance must be vectors or arrays of numbers"}
				}
				dist, err := calcEuclideanDistance(v1, v2)
				if err != nil {
					return &object.Error{Message: err.Error()}
				}
				return &object.Float{Value: dist}
			},
		},
		"manhattan_distance": {
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 2 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=2", len(args))}
				}
				v1, err1 := extractVectorSlice(args[0])
				v2, err2 := extractVectorSlice(args[1])
				if err1 != nil || err2 != nil {
					return &object.Error{Message: "arguments to manhattan_distance must be vectors or arrays of numbers"}
				}
				dist, err := calcManhattanDistance(v1, v2)
				if err != nil {
					return &object.Error{Message: err.Error()}
				}
				return &object.Float{Value: dist}
			},
		},
		"vector_search": {
			Fn: func(args ...object.Object) object.Object {
				if len(args) < 2 || len(args) > 4 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want 2 to 4", len(args))}
				}
				query, err := extractVectorSlice(args[0])
				if err != nil {
					return &object.Error{Message: fmt.Sprintf("invalid query vector: %s", err)}
				}
				candList, ok := args[1].(*object.Array)
				if !ok {
					return &object.Error{Message: "candidates must be an Array"}
				}

				topK := len(candList.Elements)
				if len(args) >= 3 {
					if intObj, ok := args[2].(*object.Integer); ok && intObj.Value > 0 {
						topK = int(intObj.Value)
					}
				}

				metric := "cosine"
				if len(args) >= 4 {
					if sObj, ok := args[3].(*object.String); ok {
						metric = sObj.Value
					}
				}

				type searchResult struct {
					item  object.Object
					score float64
				}
				var results []searchResult

				for _, cand := range candList.Elements {
					var candVec []float64
					if h, ok := cand.(*object.Hash); ok {
						for _, p := range h.Pairs {
							k := p.Key.Inspect()
							if k == "embedding" || k == "vector" || k == "vec" {
								candVec, _ = extractVectorSlice(p.Value)
								break
							}
						}
					}
					if candVec == nil {
						candVec, _ = extractVectorSlice(cand)
					}
					if candVec == nil || len(candVec) != len(query) {
						continue
					}

					var score float64
					switch metric {
					case "euclidean":
						d, _ := calcEuclideanDistance(query, candVec)
						score = d
					case "manhattan":
						d, _ := calcManhattanDistance(query, candVec)
						score = d
					default: // cosine
						s, _ := calcCosineSimilarity(query, candVec)
						score = s
					}
					results = append(results, searchResult{item: cand, score: score})
				}

				if metric == "euclidean" || metric == "manhattan" {
					sort.Slice(results, func(i, j int) bool {
						return results[i].score < results[j].score
					})
				} else {
					sort.Slice(results, func(i, j int) bool {
						return results[i].score > results[j].score
					})
				}

				if len(results) > topK {
					results = results[:topK]
				}

				outArray := make([]object.Object, len(results))
				for i, res := range results {
					pairs := make(map[object.HashKey]object.HashPair)
					itemKey := &object.String{Value: "item"}
					scoreKey := &object.String{Value: "score"}
					pairs[itemKey.HashKey()] = object.HashPair{Key: itemKey, Value: res.item}
					pairs[scoreKey.HashKey()] = object.HashPair{Key: scoreKey, Value: &object.Float{Value: res.score}}
					outArray[i] = &object.Hash{Pairs: pairs}
				}
				return &object.Array{Elements: outArray}
			},
		},
	})
}

func extractVectorSlice(obj object.Object) ([]float64, error) {
	switch o := obj.(type) {
	case *object.Vector:
		res := make([]float64, len(o.Elements))
		copy(res, o.Elements)
		return res, nil
	case *object.Array:
		res := make([]float64, len(o.Elements))
		for i, el := range o.Elements {
			v, ok := toFloat(el)
			if !ok {
				return nil, fmt.Errorf("element at index %d is not a number", i)
			}
			res[i] = v
		}
		return res, nil
	default:
		return nil, fmt.Errorf("expected Vector or Array of numbers, got %s", obj.Type())
	}
}

func calcCosineSimilarity(a, b []float64) (float64, error) {
	if len(a) != len(b) {
		return 0, fmt.Errorf("vector dimensions mismatch: %d != %d", len(a), len(b))
	}
	var dot, normA, normB float64
	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	if normA == 0 || normB == 0 {
		return 0, nil
	}
	sim := dot / (math.Sqrt(normA) * math.Sqrt(normB))
	if math.Abs(sim-1.0) < 1e-12 {
		return 1.0, nil
	}
	if math.Abs(sim-(-1.0)) < 1e-12 {
		return -1.0, nil
	}
	if sim > 1.0 {
		sim = 1.0
	} else if sim < -1.0 {
		sim = -1.0
	}
	return sim, nil
}

func calcEuclideanDistance(a, b []float64) (float64, error) {
	if len(a) != len(b) {
		return 0, fmt.Errorf("vector dimensions mismatch: %d != %d", len(a), len(b))
	}
	var sum float64
	for i := range a {
		d := a[i] - b[i]
		sum += d * d
	}
	return math.Sqrt(sum), nil
}

func calcManhattanDistance(a, b []float64) (float64, error) {
	if len(a) != len(b) {
		return 0, fmt.Errorf("vector dimensions mismatch: %d != %d", len(a), len(b))
	}
	var sum float64
	for i := range a {
		sum += math.Abs(a[i] - b[i])
	}
	return sum, nil
}
