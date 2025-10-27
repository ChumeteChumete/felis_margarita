package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"Felis_Margarita/internal/bot"
	pb "Felis_Margarita/pkg"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	grpc_retry "github.com/grpc-ecosystem/go-grpc-middleware/retry"
)

func main() {
	token := os.Getenv("TELEGRAM_TOKEN")
	if token == "" {
		log.Fatal("TELEGRAM_TOKEN not set")
	}

	grpcAddr := os.Getenv("GRPC_ADDR")
	if grpcAddr == "" {
		grpcAddr = "ml-service:50051"
	}

	conn, err := grpc.NewClient(
		grpcAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(grpc_retry.UnaryClientInterceptor(grpc_retry.WithMax(3), grpc_retry.WithBackoff(grpc_retry.BackoffExponential(100*time.Millisecond)))),
	)
	if err != nil {
		log.Fatalf("grpc connection failed: %v", err)
	}
	defer conn.Close()

	mlClient := pb.NewQnAClient(conn)
	service := bot.NewService(mlClient)
	handler := bot.NewHandler(token, service)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("shutting down...")
		cancel()
	}()

	if err := handler.Start(ctx); err != nil {
		log.Fatalf("bot crashed: %v", err)
	}
}