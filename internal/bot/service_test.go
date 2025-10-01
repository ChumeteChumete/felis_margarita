package bot

import (
	"context"
	"testing"

	pb "Felis_Margarita/pkg/proto"
	"google.golang.org/grpc"
)

// Mock ML client for testing
type mockMLClient struct {
	uploadErr error
	queryErr  error
}

func (m *mockMLClient) UploadDocument(ctx context.Context, req *pb.UploadDocRequest, opts ...grpc.CallOption) (*pb.UploadDocResponse, error) {
	if m.uploadErr != nil {
		return nil, m.uploadErr
	}
	return &pb.UploadDocResponse{
		DocId:  "test_doc_123",
		Status: "ok",
	}, nil
}

func (m *mockMLClient) Query(ctx context.Context, req *pb.QueryRequest, opts ...grpc.CallOption) (*pb.QueryResponse, error) {
	if m.queryErr != nil {
		return nil, m.queryErr
	}
	return &pb.QueryResponse{
		Answer: "test answer",
		Contexts: []*pb.Chunk{
			{
				ChunkId: "chunk1",
				Text:    "test context",
				Score:   0.95,
			},
		},
	}, nil
}

func TestUploadDocument(t *testing.T) {
	mockClient := &mockMLClient{}
	service := NewService(mockClient)

	docID, err := service.UploadDocument(
		context.Background(),
		"user123",
		"test.pdf",
		[]byte("test data"),
	)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if docID != "test_doc_123" {
		t.Errorf("expected doc_id 'test_doc_123', got '%s'", docID)
	}
}

func TestQuery(t *testing.T) {
	mockClient := &mockMLClient{}
	service := NewService(mockClient)

	resp, err := service.Query(
		context.Background(),
		"user123",
		"test question",
		5,
	)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if resp.Answer != "test answer" {
		t.Errorf("expected answer 'test answer', got '%s'", resp.Answer)
	}

	if len(resp.Contexts) != 1 {
		t.Errorf("expected 1 context, got %d", len(resp.Contexts))
	}
}