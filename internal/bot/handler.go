package bot

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	pb "Felis_Margarita/pkg/proto"
	tgbot "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"golang.org/x/sync/semaphore"
)

type Handler struct {
    bot     *tgbot.BotAPI
    service *Service
    sem     *semaphore.Weighted
}

func NewHandler(token string, service *Service) *Handler {
    bot, err := tgbot.NewBotAPI(token)
    if err != nil {
        log.Fatalf("telegram init failed: %v", err)
    }
    return &Handler{
        bot:     bot,
        service: service,
        sem:     semaphore.NewWeighted(5),
    }
}

func (h *Handler) Start(ctx context.Context) error {
    log.Println("Starting bot...")
    u := tgbot.NewUpdate(0)
    u.Timeout = 60
    updates := h.bot.GetUpdatesChan(u)

    for {
        select {
        case <-ctx.Done():
            log.Println("Stopping bot...")
            h.bot.StopReceivingUpdates()
            return nil
        case update := <-updates:
            if update.Message == nil {
                log.Println("Received empty message update")
                continue
            }
            go func() {
                defer func() {
                    if r := recover(); r != nil {
                        log.Printf("Panic in handleUpdate: %v", r)
                    }
                }()
                h.handleUpdate(ctx, update)
            }()
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
		text := update.Message.Text
		
		switch text {
		case "/start":
			h.handleStart(chatID)
			return
		case "/help":
			h.handleHelp(chatID)
			return
		case "/docs":
			h.handleListDocs(ctx, userID, chatID)
			return
		case "/clear":
			h.handleClearDocs(ctx, userID, chatID)
			return
		}
		
		// Обычный вопрос
		h.handleQuestion(ctx, userID, chatID, text, false)
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

func (h *Handler) handleQuestion(ctx context.Context, userID string, chatID int64, question string, showContexts bool) {
    log.Printf("Processing question from user %s: %s", userID, question)
    ctx, cancel := context.WithTimeout(ctx, 60*time.Second) 
    defer cancel()
    resp, err := h.service.Query(ctx, userID, question, 10)
    if err != nil {
        log.Printf("Query error for user %s: %v", userID, err)
        h.sendError(chatID, "query failed")
        return
    }
    log.Printf("Got %d contexts from ML service for user %s", len(resp.Contexts), userID)
    msg := h.formatResponse(resp, showContexts)
    log.Printf("Formatted message length: %d chars", len(msg))
    if len(msg) > 4000 {
        log.Printf("WARNING: Message too long for user %s, truncating", userID)
        msg = msg[:4000]
    }
    tmsg := tgbot.NewMessage(chatID, msg)
    tmsg.ParseMode = "HTML"
    _, err = h.bot.Send(tmsg)
    if err != nil {
        log.Printf("Failed to send message to user %s: %v", userID, err)
        return
    }
}

func (h *Handler) handleDirectQuestion(ctx context.Context, chatID int64, question string) {
	log.Printf("Direct question (no search): %s", question)
	
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	
	// Прямой запрос к ML сервису БЕЗ поиска по документам
	req := &pb.QueryRequest{
		UserId:   "direct",
		Question: question,
		TopK:     0,
	}
	
	resp, err := h.service.mlClient.DirectQuery(ctx, req)
	if err != nil {
		log.Printf("Direct query error: %v", err)
		h.sendError(chatID, "query failed")
		return
	}
	
	if resp.Answer != "" {
		h.bot.Send(tgbot.NewMessage(chatID, resp.Answer))
	} else {
		h.bot.Send(tgbot.NewMessage(chatID, "Не смог ответить, попробуй переформулировать"))
	}
}

func (h *Handler) formatResponse(resp *QueryResponse, showContexts bool) string {
	msg := ""

	if resp.Answer != "" {
		msg += resp.Answer + "\n"
	}

	// Показывает контексты только если запрошено
	if showContexts && len(resp.Contexts) > 0 {
		msg += "\n<b>📚 Источники:</b>\n"
		for i, c := range resp.Contexts {
			text := c.Text
			if len(text) > 300 {
				text = text[:300] + "..."
			}
			msg += fmt.Sprintf("%d. %s\n\n", i+1, text)
		}
	}

	return msg
}

func (h *Handler) handleStart(chatID int64) {
	msg := `🐱 Привет! Я Фелис Margarita — барханный кот в пустыне данных.

	Я работаю в двух режимах:
	📄 **С документом** — загрузи файл и спроси о нём
	💬 **Без документа** — просто общайся со мной

	Команды:
	/help — справка
	/docs — твои документы
	/clear — удалить все свои документы
	/contexts — показать источники (после ответа)

	Давай начнём! 🏜️`
		h.bot.Send(tgbot.NewMessage(chatID, msg))
	}

	func (h *Handler) handleHelp(chatID int64) {
		msg := `📚 **Помощь:**

	1️⃣ Загрузи документ (PDF, TXT, DOCX)
	2️⃣ Спроси меня о нём
	3️⃣ Я найду нужную информацию

	Или просто поговори со мной без документов!

	🔍 /contexts — показать источники ответа
	🗑️ /clear — удалить свои документы`
		h.bot.Send(tgbot.NewMessage(chatID, msg))
}

func (h *Handler) handleListDocs(ctx context.Context, userID string, chatID int64) {
	// TODO: запрос списка документов пользователя из БД
	msg := "📋 Твои документы (скоро реализуем)"
	h.bot.Send(tgbot.NewMessage(chatID, msg))
}

func (h *Handler) handleClearDocs(ctx context.Context, userID string, chatID int64) {
	// TODO: удалить все документы пользователя
	msg := "🗑️ Документы удалены (скоро реализуем)"
	h.bot.Send(tgbot.NewMessage(chatID, msg))
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
	userMsg := "❌ Что-то пошло не так.\n\n"
	
	switch errMsg {
	case "failed to get file":
		userMsg += "Не удалось получить файл. Попробуйте ещё раз."
	case "failed to download file":
		userMsg += "Ошибка загрузки файла."
	case "upload failed":
		userMsg += "Не удалось обработать документ. Проверьте формат файла."
	case "query failed":
		userMsg += "Не удалось найти ответ. Попробуйте переформулировать вопрос."
	default:
		userMsg += "Попробуйте позже или обратитесь к администратору."
	}
	
	h.bot.Send(tgbot.NewMessage(chatID, userMsg))
}