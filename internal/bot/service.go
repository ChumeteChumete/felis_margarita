package bot

import (
	"context"
	"fmt"

	pb "Felis_Margarita/pkg"
)

type Service struct {
	mlClient pb.QnAClient
}

type QueryResponse struct {
	Answer   string
	Contexts []Context
}

type Context struct {
	ChunkID string
	Text    string
	Score   float32
}

func NewService(mlClient pb.QnAClient) *Service {
	return &Service{mlClient: mlClient}
}

func (s *Service) UploadDocument(ctx context.Context, userID, filename string, data []byte) (string, error) {
	req := &pb.UploadDocRequest{
		UserId:    userID,
		Title:     filename,
		FileBytes: data,
		Filename:  filename,
	}

	resp, err := s.mlClient.UploadDocument(ctx, req)
	if err != nil {
		return "", err
	}

	return resp.DocId, nil
}

func (s *Service) Query(ctx context.Context, userID, question string, topK int32) (*QueryResponse, error) {
	req := &pb.QueryRequest{
		UserId:   userID,
		Question: question,
		TopK:     topK,
	}

	resp, err := s.mlClient.Query(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("grpc query failed: %w", err)
	}

	contexts := make([]Context, len(resp.Contexts))
	for i, c := range resp.Contexts {
		contexts[i] = Context{
			ChunkID: c.ChunkId,
			Text:    c.Text,
			Score:   c.Score,
		}
	}

	return &QueryResponse{
		Answer:   resp.Answer,
		Contexts: contexts,
	}, nil
}