package gojinn

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strings"

	"go.uber.org/zap"
)

type embeddingResponse struct {
	Data []struct {
		Embedding []float64 `json:"embedding"`
	} `json:"data"`
}

func (r *Gojinn) SemanticMatch(query string) bool {
	if strings.Contains(strings.ToLower(query), strings.ToLower(r.ToolMeta.Name)) {
		return true
	}

	if r.AIToken == "" || r.AIEndpoint == "" {
		return false
	}

	qV, err := r.getEmbedding(query)
	if err != nil {
		r.logger.Error("Failed to get query embedding", zap.Error(err))
		return false
	}

	dV, err := r.getEmbedding(r.ToolMeta.Description)
	if err != nil {
		r.logger.Error("Failed to get description embedding", zap.Error(err))
		return false
	}

	similarity := cosineSimilarity(qV, dV)

	r.logger.Debug("Semantic routing check",
		zap.String("query", query),
		zap.Float64("similarity", similarity))

	return similarity > 0.75
}

func (r *Gojinn) getEmbedding(text string) ([]float64, error) {
	if val, ok := r.aiCache.Load("emb_" + text); ok {
		return val.([]float64), nil
	}

	reqBody, _ := json.Marshal(map[string]interface{}{
		"input": text,
		"model": "text-embedding-3-small",
	})

	req, _ := http.NewRequest("POST", r.AIEndpoint+"/embeddings", bytes.NewBuffer(reqBody))
	req.Header.Set("Authorization", "Bearer "+r.AIToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var embResp embeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&embResp); err != nil {
		return nil, err
	}

	if len(embResp.Data) > 0 {
		r.aiCache.Store("emb_"+text, embResp.Data[0].Embedding)
		return embResp.Data[0].Embedding, nil
	}

	return nil, fmt.Errorf("no embedding returned")
}

func cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) {
		return 0.0
	}
	var dotProduct, normA, normB float64
	for i := 0; i < len(a); i++ {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	if normA == 0 || normB == 0 {
		return 0.0
	}
	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

type jsonRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      interface{}     `json:"id"`
}

type jsonRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	Result  interface{} `json:"result,omitempty"`
	Error   interface{} `json:"error,omitempty"`
	ID      interface{} `json:"id"`
}

func (r *Gojinn) ServeMCP(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "event: endpoint\ndata: %s/message\n\n", req.URL.Path)
	flusher.Flush()

	<-req.Context().Done()
}

func (r *Gojinn) HandleMCPMessage(w http.ResponseWriter, req *http.Request) {
	var jsonReq jsonRPCRequest
	if err := json.NewDecoder(req.Body).Decode(&jsonReq); err != nil {
		return
	}

	var response jsonRPCResponse
	response.JSONRPC = "2.0"
	response.ID = jsonReq.ID

	switch jsonReq.Method {
	case "tools/list":
		if r.ExposeAsTool {
			tool := ToolDefinition{
				Name:        r.ToolMeta.Name,
				Description: r.ToolMeta.Description,
				InputSchema: r.ToolMeta.InputSchema,
			}
			response.Result = map[string]interface{}{"tools": []ToolDefinition{tool}}
		}
	case "tools/call":
		var params struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments"`
		}
		json.Unmarshal(jsonReq.Params, &params)
		if params.Name == r.ToolMeta.Name {
			payload, _ := json.Marshal(params.Arguments)
			result, err := r.runSyncJob(req.Context(), r.Path, string(payload))
			if err != nil {
				response.Error = map[string]string{"message": err.Error()}
			} else {
				response.Result = map[string]interface{}{
					"content": []map[string]string{{"type": "text", "text": result}},
				}
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

type ToolDefinition struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema interface{} `json:"inputSchema"`
}
