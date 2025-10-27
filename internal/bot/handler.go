package bot

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	pb "Felis_Margarita/pkg"
	tgbot "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"golang.org/x/sync/semaphore"
)

// userProcessing хранит ID пользователей, которые сейчас в обработке (только offline)
var (
	userProcessing = make(map[string]bool)
	procMu         sync.RWMutex
)

func isUserProcessing(userID string) bool {
	procMu.RLock()
	defer procMu.RUnlock()
	return userProcessing[userID]
}

func setUserProcessing(userID string, processing bool) {
	procMu.Lock()
	defer procMu.Unlock()
	if processing {
		userProcessing[userID] = true
	} else {
		delete(userProcessing, userID)
	}
}

// UserMode хранит режим пользователя
type UserMode string

const (
	ModeOffline UserMode = "offline"
	ModeOnline  UserMode = "online"
)

// userModes — временный in-memory storage (заменить на Redis позже)
var (
	userModes = make(map[string]UserMode)
	mu        sync.RWMutex
)

func getUserMode(userID string) UserMode {
	mu.RLock()
	defer mu.RUnlock()
	if mode, ok := userModes[userID]; ok {
		return mode
	}
	return ModeOffline // по умолчанию
}

func setUserMode(userID string, mode UserMode) {
	mu.Lock()
	defer mu.Unlock()
	userModes[userID] = mode
}

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
		case "/online":
			h.handleSetMode(ctx, userID, chatID, "online")
			return
		case "/offline":
			h.handleSetMode(ctx, userID, chatID, "offline")
			return
		case "/docs":
			h.handleListDocs(ctx, userID, chatID)
			return
		case "/clear":
			h.handleClearDocs(ctx, userID, chatID)
			return
		}

		mode := getUserMode(userID)
		if mode == ModeOnline {
			h.handleDirectQuestion(ctx, chatID, text)
		} else {
			h.handleQuestion(ctx, userID, chatID, text, false)
		}
	}
}

func (h *Handler) handleSetMode(ctx context.Context, userID string, chatID int64, mode string) {
	req := &pb.SetModeRequest{Mode: mode}
	_, err := h.service.mlClient.SetMode(ctx, req)
	if err != nil {
		h.bot.Send(tgbot.NewMessage(chatID, "❌ Не удалось переключить режим"))
		return
	}

	if mode == "online" {
		setUserMode(userID, ModeOnline)
		msg := "✅ Режим переключён на онлайн\n\n" +
			"Теперь я свободен, как ветер в пустыне! " +
			"Твои документы недоступны — я не вижу их и не помню. " +
			"Просто задавай вопросы, и я отвечу как настоящий барханный кот 🐾"
		h.bot.Send(tgbot.NewMessage(chatID, msg))
	} else {
		setUserMode(userID, ModeOffline)
		msg := "✅ Режим переключён на оффлайн\n\n" +
			"Теперь я вижу твои документы и отвечаю строго по ним. " +
			"Все данные остаются со мной — интернет запрещён! 🏜️"
		h.bot.Send(tgbot.NewMessage(chatID, msg))
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
	// Проверяем, не в обработке ли уже
	if isUserProcessing(userID) {
		h.bot.Send(tgbot.NewMessage(chatID, "🐾 Подожди, я ещё ищу ответ на твой предыдущий вопрос..."))
		return
	}

	// Устанавливаем статус "в обработке"
	setUserProcessing(userID, true)
	defer setUserProcessing(userID, false)

	log.Printf("Processing question (offline mode) from user %s: %s", userID, question)

	// Отправляем "печатает..." — анимация через редактирование
	typingMsg := tgbot.NewMessage(chatID, "🐾 Фелис услышал тебя. Ищу нужную информацию в твоих документах...")
	sentMsg, err := h.bot.Send(typingMsg)
	if err != nil {
		log.Printf("Failed to send typing indicator: %v", err)
	}

	// Таймаут 5 минут
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	resp, err := h.service.Query(ctx, userID, question, 2)
	if err != nil {
		log.Printf("Query error for user %s: %v", userID, err)
		// Редактируем на ошибку
		if sentMsg.MessageID != 0 {
			editMsg := tgbot.NewEditMessageText(chatID, sentMsg.MessageID, "❌ Не удалось найти ответ. Попробуй переформулировать вопрос.")
			h.bot.Send(editMsg)
		} else {
			h.sendError(chatID, "query failed")
		}
		return
	}

	msg := h.formatResponse(resp, showContexts)
	if len(msg) > 4000 {
		msg = msg[:4000]
	}

	if sentMsg.MessageID != 0 {
		editMsg := tgbot.NewEditMessageText(chatID, sentMsg.MessageID, msg)
		editMsg.ParseMode = "HTML"
		h.bot.Send(editMsg)
	} else {
		tmsg := tgbot.NewMessage(chatID, msg)
		tmsg.ParseMode = "HTML"
		h.bot.Send(tmsg)
	}
}

func (h *Handler) handleDirectQuestion(ctx context.Context, chatID int64, question string) {
	log.Printf("Direct question (online mode): %s", question)

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

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
	req := &pb.ListDocsRequest{UserId: userID}
	resp, err := h.service.mlClient.ListDocuments(ctx, req)
	if err != nil {
		log.Printf("ListDocs error for user %s: %v", userID, err)
		h.bot.Send(tgbot.NewMessage(chatID, "❌ Не удалось загрузить список документов"))
		return
	}

	if len(resp.Titles) == 0 {
		h.bot.Send(tgbot.NewMessage(chatID, "📂 У тебя пока нет документов."))
		return
	}

	msg := "📋 Твои документы:\n"
	for i, title := range resp.Titles {
		msg += fmt.Sprintf("%d. %s\n", i+1, title)
	}
	h.bot.Send(tgbot.NewMessage(chatID, msg))
}

func (h *Handler) handleClearDocs(ctx context.Context, userID string, chatID int64) {
	req := &pb.ClearDocsRequest{UserId: userID}
	_, err := h.service.mlClient.ClearDocuments(ctx, req)
	if err != nil {
		log.Printf("ClearDocs error for user %s: %v", userID, err)
		h.bot.Send(tgbot.NewMessage(chatID, "❌ Не удалось удалить документы"))
		return
	}

	h.bot.Send(tgbot.NewMessage(chatID, "🗑️ Все твои документы удалены. Пустыня чиста!"))
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