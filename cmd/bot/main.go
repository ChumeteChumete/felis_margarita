package main

import (
	"context"
	"log"
	"os"
	"strconv"
	"time"

	tgbot "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"google.golang.org/grpc"

	pb "Felis_Margarita/ml_service/proto"
)

func main() {
	token := os.Getenv("TELEGRAM_TOKEN")
	if token == "" {
		log.Fatal("TELEGRAM_TOKEN required")
	}
	grpcAddr := os.Getenv("GRPC_ADDR")
	if grpcAddr == "" {
		grpcAddr = "localhost:50051"
	}

	// gRPC connection to ML service
	conn, err := grpc.Dial(grpcAddr, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("failed connect grpc: %v", err)
	}
	defer conn.Close()
	client := pb.NewQnAClient(conn)

	bot, err := tgbot.NewBotAPI(token)
	if err != nil {
		log.Fatalf("telegram init: %v", err)
	}
	bot.Debug = false

	u := tgbot.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	ctx := context.Background()

	for update := range updates {
		if update.Message == nil {
			continue
		}

		uid := strconv.FormatInt(update.Message.From.ID, 10)

		// если прислали документ — скачиваем и отправляем в ML
		if update.Message.Document != nil {
			doc := update.Message.Document
			fileID := doc.FileID
			file, err := bot.GetFile(tgbot.FileConfig{FileID: fileID})
			if err != nil {
				bot.Send(tgbot.NewMessage(update.Message.Chat.ID, "ошибка получения файла"))
				continue
			}
			// скачиваем содержимое файла (напрямую через Telegram API)
			url := file.Link(bot.Token)
			// скачиваем байты
			data, err := downloadFile(url)
			if err != nil {
				bot.Send(tgbot.NewMessage(update.Message.Chat.ID, "ошибка скачивания файла"))
				continue
			}
			req := &pb.UploadDocRequest{
				UserId:    uid,
				Title:     doc.FileName,
				FileBytes: data,
				Filename:  doc.FileName,
			}
			resp, err := client.UploadDocument(ctx, req)
			if err != nil {
				bot.Send(tgbot.NewMessage(update.Message.Chat.ID, "ошибка на ML сервисе"))
				continue
			}
			bot.Send(tgbot.NewMessage(update.Message.Chat.ID, "Документ загружен: "+resp.DocId))
			continue
		}

		// если просто текст — воспринимаем как вопрос
		if update.Message.Text != "" {
			// короткая команда /ask вопрос или просто текст
			question := update.Message.Text
			req := &pb.QueryRequest{
				UserId:  uid,
				Question: question,
				TopK:    5,
			}
			// контекст и ответ
			ctx2, cancel := context.WithTimeout(ctx, 10*time.Second)
			resp, err := client.Query(ctx2, req)
			cancel()
			if err != nil {
				bot.Send(tgbot.NewMessage(update.Message.Chat.ID, "ошибка запроса: "+err.Error()))
				continue
			}
			// формируем ответ: показываем contexts и answer
			msg := ""
			if resp.Answer != "" {
				msg += "*Answer:*\n" + resp.Answer + "\n\n"
			}
			msg += "*Contexts:*\n"
			for i, c := range resp.Contexts {
				msg += strconv.Itoa(i+1) + ". " + truncate(c.Text, 500) + "\n\n"
			}
			tmsg := tgbot.NewMessage(update.Message.Chat.ID, msg)
			tmsg.ParseMode = "Markdown"
			bot.Send(tmsg)
		}
	}
}
