package bot

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"

	tgbot "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Handler struct {
	bot     *tgbot.BotAPI
	service *Service
}

func NewHandler(token string, service *Service) *Handler {
	bot, err := tgbot.NewBotAPI(token)
	if err != nil {
		log.Fatalf("telegram init failed: %v", err)
	}
	return &Handler{
		bot:     bot,
		service: service,
	}
}

func (h *Handler) Start(ctx context.Context) error {
	u := tgbot.NewUpdate(0)
	u.Timeout = 60
	updates := h.bot.GetUpdatesChan(u)

	for {
		select {
		case <-ctx.Done():
			h.bot.StopReceivingUpdates()
			return nil
		case update := <-updates:
			if update.Message == nil {
				continue
			}
			go h.handleUpdate(ctx, update)
		}
	}
}

func (h *Handler) handleUpdate(ctx context.Context, update tgbot.Update) {
	userID := strconv.FormatInt(update.Message.From.ID, 10)
	chatID := update.Message.Chat.ID

	if update.Message.Document != nil {
		h.handleDocument(ctx, userID, chatID, update.Message.Document)
		return
	}

	if update.Message.Text != "" {
		h.handleQuestion(ctx, userID, chatID, update.Message.Text)
		return
	}
}

func (h *Handler) handleDocument(ctx context.Context, userID string, chatID int64, doc *tgbot.Document) {
	file, err := h.bot.GetFile(tgbot.FileConfig{FileID: doc.FileID})
	if err != nil {
		h.sendError(chatID, "failed to get file")
		return
	}

	data, err := h.downloadFile(file.Link(h.bot.Token))
	if err != nil {
		h.sendError(chatID, "failed to download file")
		return
	}

	docID, err := h.service.UploadDocument(ctx, userID, doc.FileName, data)
	if err != nil {
		h.sendError(chatID, "upload failed")
		return
	}

	h.bot.Send(tgbot.NewMessage(chatID, fmt.Sprintf("Document uploaded: %s", docID)))
}

func (h *Handler) handleQuestion(ctx context.Context, userID string, chatID int64, question string) {
	resp, err := h.service.Query(ctx, userID, question, 5)
	if err != nil {
		h.sendError(chatID, "query failed")
		return
	}

	msg := h.formatResponse(resp)
	tmsg := tgbot.NewMessage(chatID, msg)
	tmsg.ParseMode = "Markdown"
	h.bot.Send(tmsg)
}

func (h *Handler) formatResponse(resp *QueryResponse) string {
	msg := ""
	if resp.Answer != "" {
		msg += "*Answer:*\n" + resp.Answer + "\n\n"
	}
	msg += "*Contexts:*\n"
	for i, c := range resp.Contexts {
		text := c.Text
		if len(text) > 500 {
			text = text[:500] + "..."
		}
		msg += fmt.Sprintf("%d. %s\n\n", i+1, text)
	}
	return msg
}

func (h *Handler) downloadFile(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func (h *Handler) sendError(chatID int64, errMsg string) {
	h.bot.Send(tgbot.NewMessage(chatID, "Error: "+errMsg))
}