package handler

import "context"

type HealthResponse struct {
	Body struct {
		Status  string `json:"status" example:"ok"`
		Service string `json:"service" example:"ZenReply API"`
		Version string `json:"version" example:"1.0.0"`
	}
}

func (h *Handler) HealthCheck(ctx context.Context, input *struct{}) (*HealthResponse, error) {
	resp := &HealthResponse{}
	resp.Body.Status = "ok"
	resp.Body.Service = "ZenReply API"
	resp.Body.Version = "1.0.0"

	return resp, nil
}
