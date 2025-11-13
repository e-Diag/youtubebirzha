package handlers

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"youtube-market/internal/db"
	"youtube-market/internal/models"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	managerHelpLink        = "@birzha_manager"
	commandNewAd           = "/newad"
	commandAdDetails       = "/ad"
	commandCancel          = "/cancel"
	sessionTimeoutDuration = 30 * time.Minute
)

// Русские названия для категорий
var categoryLabels = map[string]string{
	"services": "Услуги",
	"buysell":  "Купля/Продажа",
	"other":    "Другое",
}

var categoryValues = map[string]string{
	"Услуги":        "services",
	"Купля/Продажа": "buysell",
	"Другое":        "other",
}

// Русские названия для режимов
var modeLabels = map[string]map[string]string{
	"services": {
		"offer":  "Предлагаю услугу",
		"search": "Ищу услугу",
	},
	"buysell": {
		"sell": "Продаю",
		"buy":  "Покупаю",
	},
	"other": {
		"general": "Объявление",
	},
}

var modeValues = map[string]map[string]string{
	"services": {
		"Предлагаю услугу": "offer",
		"Ищу услугу":       "search",
	},
	"buysell": {
		"Продаю":  "sell",
		"Покупаю": "buy",
	},
	"other": {
		"Объявление": "general",
	},
}

// Русские названия для тегов
var tagLabels = map[string]map[string]string{
	"services": {
		"all":      "Все",
		"designer": "Дизайнер",
		"script":   "Сценарист",
		"voice":    "Озвучивание",
		"other":    "Другое",
	},
	"buysell": {
		"all":       "Все",
		"konechka":  "Конечка",
		"channel":   "Канал",
		"video":     "Видео",
		"adsense":   "Адсенс",
		"templates": "Шаблоны",
	},
	"other": {
		"all":       "Все",
		"education": "Обучение",
		"courses":   "Курсы",
		"cheats":    "Читы",
		"mods":      "Моды",
		"niche":     "Ниша",
		"schemes":   "Схемы",
		"boost":     "Накрутка",
	},
}

var tagValues = map[string]map[string]string{
	"services": {
		"Все":         "all",
		"Дизайнер":    "designer",
		"Сценарист":   "script",
		"Озвучивание": "voice",
		"Другое":      "other",
	},
	"buysell": {
		"Все":     "all",
		"Конечка": "konechka",
		"Канал":   "channel",
		"Видео":   "video",
		"Адсенс":  "adsense",
		"Шаблоны": "templates",
	},
	"other": {
		"Все":      "all",
		"Обучение": "education",
		"Курсы":    "courses",
		"Читы":     "cheats",
		"Моды":     "mods",
		"Ниша":     "niche",
		"Схемы":    "schemes",
		"Накрутка": "boost",
	},
}

type conversationStage int

const (
	stageNone conversationStage = iota
	stageAwaitAction
	stageAwaitPhoto
	stageAwaitTitle
	stageAwaitDescription
	stageAwaitUsername
	stageAwaitCategory
	stageAwaitMode
	stageAwaitTag
	stageAwaitDuration
	stageAwaitPremium
	stageAwaitUserId
	stageAwaitConfirmation
	stageAwaitRenewDuration
	stageAwaitBlacklistAction
	stageAwaitBlacklistAdd
	stageAwaitBlacklistRemove
	stageAwaitFindAdID
	stageAwaitSelectAd
)

type adOperation int

const (
	opCreate adOperation = iota
	opEdit
	opRenew
)

type adSession struct {
	Operation     adOperation
	Stage         conversationStage
	Ad            models.Ad
	DurationDays  int
	LastActivity  time.Time
	ChatID        int64
	BotMessageIDs []int // ID сообщений бота для удаления
}

var (
	sessionRegistry = struct {
		sync.Mutex
		data map[int64]*adSession
	}{data: make(map[int64]*adSession)}
)

// parseManagerIDs парсит строку с ID менеджеров (формат: "ID1,ID2,ID3")
func parseManagerIDs(managerIDsStr string) ([]int64, error) {
	if managerIDsStr == "" {
		return nil, fmt.Errorf("MANAGER_ID is empty")
	}

	parts := strings.Split(managerIDsStr, ",")
	ids := make([]int64, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		id, err := strconv.ParseInt(part, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid manager ID: %s", part)
		}
		if id == 0 {
			continue
		}
		ids = append(ids, id)
	}

	if len(ids) == 0 {
		return nil, fmt.Errorf("no valid manager IDs found")
	}

	return ids, nil
}

// isManager проверяет, является ли пользователь менеджером
func isManager(userID int64, managerIDs []int64) bool {
	for _, managerID := range managerIDs {
		if userID == managerID {
			return true
		}
	}
	return false
}

func RunManagerBot() {
	botToken := os.Getenv("BOT_TOKEN")
	if botToken == "" {
		log.Println("BOT_TOKEN not set, manager bot disabled")
		return
	}

	managerIDsStr := os.Getenv("MANAGER_ID")
	managerIDs, err := parseManagerIDs(managerIDsStr)
	if err != nil {
		log.Printf("MANAGER_ID not set or invalid (%v), manager bot disabled", err)
		return
	}

	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Fatal("bot init failed:", err)
	}

	setBotToken(botToken)
	startAdSchedulers(bot)

	log.Printf("Manager bot started for user IDs: %v", managerIDs)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		switch {
		case update.Message != nil:
			handleManagerMessage(bot, managerIDs, update.Message)
		case update.CallbackQuery != nil:
			handleCallbackQuery(bot, managerIDs, update.CallbackQuery)
		}
	}
}

func handleManagerMessage(bot *tgbotapi.BotAPI, managerIDs []int64, msg *tgbotapi.Message) {
	if msg.From == nil || !isManager(msg.From.ID, managerIDs) {
		return
	}

	// Удаляем сообщение менеджера с задержкой (не сразу)
	go func() {
		time.Sleep(5 * time.Second)
		deleteMessage(bot, msg.Chat.ID, msg.MessageID)
	}()

	// Обработка пересланных сообщений от пользователей (для получения ID) - проверяем ПЕРВЫМ
	if msg.ForwardFrom != nil {
		handleForwardedMessage(bot, msg)
		return
	}

	text := strings.TrimSpace(msg.Text)
	if text == "" && msg.Photo == nil {
		return
	}

	// Команды
	if strings.EqualFold(text, "/start") || strings.EqualFold(text, "/menu") {
		showMainMenu(bot, msg.Chat.ID)
		return
	}

	if isCommand(text, commandNewAd) {
		startCreateSession(bot, msg.Chat.ID)
		return
	}

	// Обработка текстового ввода в активной сессии
	if session := getSession(msg.Chat.ID); session != nil && session.Stage != stageNone {
		handleSessionInput(bot, msg, session)
		return
	}

	// Если нет активной сессии, показываем меню
	showMainMenu(bot, msg.Chat.ID)
}

// handleForwardedMessage обрабатывает пересланные сообщения для получения ID пользователя
func handleForwardedMessage(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	// Проверяем, что сообщение действительно переслано
	if msg.ForwardFrom == nil {
		log.Printf("Ошибка: ForwardFrom == nil")
		sendText(bot, msg.Chat.ID, "❌ Не удалось получить информацию о пользователе. Убедитесь, что пользователь разрешил пересылку сообщений.")
		return
	}

	userID := msg.ForwardFrom.ID
	username := msg.ForwardFrom.UserName

	if userID == 0 {
		log.Printf("Ошибка: ForwardFrom.ID == 0, возможно пользователь скрыл свой ID")
		// Пробуем получить ID из ForwardFromChat (для каналов/групп)
		if msg.ForwardFromChat != nil && msg.ForwardFromChat.ID != 0 {
			userID = msg.ForwardFromChat.ID
			log.Printf("Получен ID из ForwardFromChat: %d", userID)
		} else {
			sendText(bot, msg.Chat.ID, "❌ Не удалось получить ID пользователя. Убедитесь, что пользователь разрешил пересылку сообщений.")
			return
		}
	}

	log.Printf("Обработка пересланного сообщения: UserID=%d, Username=%s", userID, username)

	session := getSession(msg.Chat.ID)

	// Если нет активной сессии, создаём временную для поиска
	if session == nil {
		session = &adSession{
			Stage:         stageAwaitFindAdID,
			LastActivity:  time.Now(),
			ChatID:        msg.Chat.ID,
			BotMessageIDs: []int{},
		}
		setSession(msg.Chat.ID, session)
	}

	clientID := strconv.FormatInt(userID, 10)

	log.Printf("Получено пересланное сообщение: UserID=%d, Username=%s, ClientID=%s, Stage=%d", userID, username, clientID, session.Stage)

	// Если мы ожидаем ID пользователя (при создании объявления)
	if session.Stage == stageAwaitUserId {
		// Устанавливаем ID
		session.Ad.ClientID = clientID
		session.Ad.UserID = userID

		// Устанавливаем username из пересланного сообщения, если он есть
		if username != "" {
			session.Ad.Username = username
			log.Printf("Username получен из пересланного сообщения: %s", username)
			sendText(bot, msg.Chat.ID, fmt.Sprintf("✅ ID пользователя получен: %d\n✅ Username: @%s", userID, username))
			session.Stage = stageAwaitCategory
			showCategoryPrompt(bot, msg.Chat.ID, session)
		} else {
			// Если username нет, запрашиваем его отдельно
			log.Printf("Username не найден в пересланном сообщении, запрашиваем отдельно")
			keyboard := tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("⏭ Пропустить (без username)", "skip_username"),
				),
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("◀️ Назад", getBackCallback(session)),
				),
			)
			msgText := tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("✅ ID пользователя получен: %d\n\n👤 *Введите username для контакта* (например: @username)\n\nИли нажмите \"Пропустить\", если username не нужен.", userID))
			msgText.ParseMode = "Markdown"
			msgText.ReplyMarkup = keyboard
			sentMsg, err := bot.Send(msgText)
			if err == nil {
				addBotMessage(msg.Chat.ID, sentMsg.MessageID)
			}
			session.Stage = stageAwaitUsername
		}
		return
	}

	// Если мы ищем объявление и получили пересланное сообщение
	if session.Stage == stageAwaitFindAdID {
		// Ищем все объявления по ClientID
		var ads []models.Ad
		if err := db.DB.Where("client_id = ?", clientID).Order("created_at DESC").Find(&ads).Error; err != nil {
			sendText(bot, msg.Chat.ID, "❌ Ошибка при поиске объявлений.")
			return
		}

		if len(ads) == 0 {
			keyboard := tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("◀️ Назад", "menu_main"),
				),
			)
			msgText := tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("❌ Объявления для пользователя ID %d не найдены.", userID))
			msgText.ReplyMarkup = keyboard
			sentMsg, err := bot.Send(msgText)
			if err == nil {
				addBotMessage(msg.Chat.ID, sentMsg.MessageID)
				go scheduleDeletePreviousMessages(bot, msg.Chat.ID, session, sentMsg.MessageID)
			}
			clearSession(msg.Chat.ID)
			return
		}

		// Показываем результаты
		handleFindAdResults(bot, msg.Chat.ID, ads, session)
		return
	}

	// Если сессия есть, но этап не подходит - показываем сообщение
	sendText(bot, msg.Chat.ID, fmt.Sprintf("✅ Получен ID пользователя: %d\n\nДля создания объявления используйте /newad", userID))
}

func handleCallbackQuery(bot *tgbotapi.BotAPI, managerIDs []int64, callback *tgbotapi.CallbackQuery) {
	if callback.From == nil || !isManager(callback.From.ID, managerIDs) {
		return
	}

	// Подтверждаем callback
	bot.Request(tgbotapi.NewCallback(callback.ID, ""))

	data := callback.Data
	chatID := callback.Message.Chat.ID

	// Получаем сессию для отслеживания сообщений
	session := getSession(chatID)

	// Сохраняем ID текущего сообщения для последующего удаления
	currentMsgID := 0
	if callback.Message != nil {
		currentMsgID = callback.Message.MessageID
		// Добавляем текущее сообщение в список для последующего удаления
		if session != nil {
			addBotMessage(chatID, currentMsgID)
		}
	}

	// Обработка callback данных
	// НЕ удаляем сообщения здесь - удаление происходит только после отправки нового сообщения
	switch {
	case data == "menu_main":
		showMainMenu(bot, chatID)
	case data == "menu_new_ad":
		startCreateSession(bot, chatID)
	case data == "menu_find_ad":
		startFindAdSession(bot, chatID)
	case data == "menu_blacklist":
		showBlacklistMenu(bot, chatID)
	case data == "blacklist_view":
		showBlacklist(bot, chatID)
	case data == "blacklist_add":
		startBlacklistAdd(bot, chatID)
	case data == "blacklist_remove":
		startBlacklistRemove(bot, chatID)
	case strings.HasPrefix(data, "ad_action_"):
		handleAdActionCallback(bot, chatID, data)
	case data == "category_edit":
		handleEditSetting(bot, chatID, "category")
	case data == "mode_edit":
		handleEditSetting(bot, chatID, "mode")
	case data == "tag_edit":
		handleEditSetting(bot, chatID, "tag")
	case data == "duration_edit":
		handleEditSetting(bot, chatID, "duration")
	case data == "premium_edit":
		handleEditSetting(bot, chatID, "premium")
	case strings.HasPrefix(data, "category_"):
		handleCategoryCallback(bot, chatID, data)
	case strings.HasPrefix(data, "mode_"):
		handleModeCallback(bot, chatID, data)
	case strings.HasPrefix(data, "tag_"):
		handleTagCallback(bot, chatID, data)
	case strings.HasPrefix(data, "duration_"):
		handleDurationCallback(bot, chatID, data)
	case strings.HasPrefix(data, "premium_"):
		handlePremiumCallback(bot, chatID, data)
	case data == "save_from_settings":
		handleSaveFromSettings(bot, chatID)
	case data == "confirm_yes":
		handleConfirmYes(bot, chatID)
	case data == "confirm_no":
		handleConfirmNo(bot, chatID)
	case data == "back":
		handleBack(bot, chatID)
	case data == "skip_photo":
		handleSkipPhoto(bot, chatID)
	case data == "skip_user_id":
		handleSkipUserID(bot, chatID)
	case data == "skip_username":
		handleSkipUsername(bot, chatID)
	case strings.HasPrefix(data, "renew_duration_"):
		handleRenewDurationCallback(bot, chatID, data)
	case data == "ad_edit":
		handleAdEdit(bot, chatID)
	case data == "ad_renew":
		handleAdRenew(bot, chatID)
	case data == "ad_remove":
		handleAdRemove(bot, chatID)
	case data == "ad_publish":
		handleAdPublish(bot, chatID)
	case strings.HasPrefix(data, "select_ad_"):
		handleSelectAd(bot, chatID, data)
	case data == "edit_after_preview":
		handleEditAfterPreview(bot, chatID)
	}
}

func showMainMenu(bot *tgbotapi.BotAPI, chatID int64) {
	clearSession(chatID)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("➕ Создать объявление", "menu_new_ad"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔍 Найти объявление", "menu_find_ad"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🚫 Чёрный список", "menu_blacklist"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, "📋 *Меню менеджера*\n\nВыберите действие:")
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard

	sentMsg, err := bot.Send(msg)
	if err == nil {
		addBotMessage(chatID, sentMsg.MessageID)
		// Удаляем предыдущие сообщения после отправки нового меню
		session := getSession(chatID)
		if session != nil {
			go scheduleDeletePreviousMessages(bot, chatID, session, sentMsg.MessageID)
		}
	}
}

func showBlacklistMenu(bot *tgbotapi.BotAPI, chatID int64) {
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📋 Просмотр", "blacklist_view"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("➕ Добавить", "blacklist_add"),
			tgbotapi.NewInlineKeyboardButtonData("➖ Удалить", "blacklist_remove"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("◀️ Назад", "menu_main"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, "🚫 *Управление чёрным списком*")
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard

	sentMsg, err := bot.Send(msg)
	if err == nil {
		addBotMessage(chatID, sentMsg.MessageID)
		// Удаляем предыдущие сообщения после отправки нового
		session := getSession(chatID)
		if session != nil {
			go scheduleDeletePreviousMessages(bot, chatID, session, sentMsg.MessageID)
		}
	}
}

func showBlacklist(bot *tgbotapi.BotAPI, chatID int64) {
	var scammers []models.User
	if err := db.DB.Where("is_scammer = ?", true).Order("username ASC").Find(&scammers).Error; err != nil {
		sendText(bot, chatID, "Ошибка загрузки чёрного списка.")
		return
	}

	if len(scammers) == 0 {
		text := "📋 *Чёрный список пуст*"
		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("◀️ Назад", "menu_blacklist"),
			),
		)
		msg := tgbotapi.NewMessage(chatID, text)
		msg.ParseMode = "Markdown"
		msg.ReplyMarkup = keyboard
		sentMsg, err := bot.Send(msg)
		if err == nil {
			addBotMessage(chatID, sentMsg.MessageID)
			// Удаляем предыдущие сообщения после отправки нового
			session := getSession(chatID)
			if session != nil {
				go scheduleDeletePreviousMessages(bot, chatID, session, sentMsg.MessageID)
			}
		}
		return
	}

	var text strings.Builder
	text.WriteString("📋 *Чёрный список:*\n\n")
	for i, user := range scammers {
		if i >= 50 { // Ограничение Telegram на длину сообщения
			text.WriteString(fmt.Sprintf("\n... и ещё %d пользователей", len(scammers)-50))
			break
		}
		text.WriteString(fmt.Sprintf("• @%s\n", user.Username))
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("◀️ Назад", "menu_blacklist"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, text.String())
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard

	sentMsg, err := bot.Send(msg)
	if err == nil {
		addBotMessage(chatID, sentMsg.MessageID)
		// Удаляем предыдущие сообщения после отправки нового
		session := getSession(chatID)
		if session != nil {
			go scheduleDeletePreviousMessages(bot, chatID, session, sentMsg.MessageID)
		}
	}
}

func startBlacklistAdd(bot *tgbotapi.BotAPI, chatID int64) {
	session := &adSession{
		Stage:        stageAwaitBlacklistAdd,
		LastActivity: time.Now(),
		ChatID:       chatID,
	}
	setSession(chatID, session)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("◀️ Назад", "menu_blacklist"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, "➕ *Добавить в чёрный список*\n\nОтправьте username (например: @username)")
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard

	sentMsg, err := bot.Send(msg)
	if err == nil {
		addBotMessage(chatID, sentMsg.MessageID)
	}
}

func startBlacklistRemove(bot *tgbotapi.BotAPI, chatID int64) {
	session := &adSession{
		Stage:        stageAwaitBlacklistRemove,
		LastActivity: time.Now(),
		ChatID:       chatID,
	}
	setSession(chatID, session)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("◀️ Назад", "menu_blacklist"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, "➖ *Удалить из чёрного списка*\n\nОтправьте username (например: @username)")
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard

	sentMsg, err := bot.Send(msg)
	if err == nil {
		addBotMessage(chatID, sentMsg.MessageID)
		// Удаляем предыдущие сообщения после отправки нового
		session := getSession(chatID)
		if session != nil {
			go scheduleDeletePreviousMessages(bot, chatID, session, sentMsg.MessageID)
		}
	}
}

func startFindAdSession(bot *tgbotapi.BotAPI, chatID int64) {
	session := &adSession{
		Stage:         stageAwaitFindAdID,
		LastActivity:  time.Now(),
		ChatID:        chatID,
		BotMessageIDs: []int{},
	}
	setSession(chatID, session)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("◀️ Назад", "menu_main"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, "🔍 *Найти объявления*\n\nОтправьте ID клиента (только цифры) или перешлите любое сообщение от пользователя:")
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard

	sentMsg, err := bot.Send(msg)
	if err == nil {
		addBotMessage(chatID, sentMsg.MessageID)
		// Удаляем предыдущие сообщения после отправки нового
		session := getSession(chatID)
		if session != nil {
			go scheduleDeletePreviousMessages(bot, chatID, session, sentMsg.MessageID)
		}
	}
}

func startCreateSession(bot *tgbotapi.BotAPI, chatID int64) {
	session := &adSession{
		Operation:     opCreate,
		Stage:         stageAwaitPhoto,
		LastActivity:  time.Now(),
		ChatID:        chatID,
		BotMessageIDs: []int{},
		Ad:            models.Ad{},
	}
	setSession(chatID, session)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⏭ Пропустить", "skip_photo"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("◀️ Отмена", "menu_main"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, "📸 *Шаг 1: Фото*\n\nОтправьте фото объявления или пропустите этот шаг.")
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard

	sentMsg, err := bot.Send(msg)
	if err == nil {
		addBotMessage(chatID, sentMsg.MessageID)
		// Удаляем предыдущие сообщения после отправки нового
		go scheduleDeletePreviousMessages(bot, chatID, session, sentMsg.MessageID)
	}
}

func handleAdActionCallback(bot *tgbotapi.BotAPI, chatID int64, data string) {
	// Обработка действий с объявлением (если нужно)
}

func handleAdEdit(bot *tgbotapi.BotAPI, chatID int64) {
	session := getSession(chatID)
	if session == nil {
		return
	}

	session.Operation = opEdit
	session.Stage = stageAwaitPhoto

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⏭ Пропустить", "skip_photo"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("◀️ Назад", fmt.Sprintf("ad_action_%d", session.Ad.ID)),
		),
	)

	msg := tgbotapi.NewMessage(chatID, "📸 *Шаг 1: Фото*\n\nОтправьте новое фото или пропустите этот шаг.")
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard

	sentMsg, err := bot.Send(msg)
	if err == nil {
		addBotMessage(chatID, sentMsg.MessageID)
	}
}

func handleAdRenew(bot *tgbotapi.BotAPI, chatID int64) {
	session := getSession(chatID)
	if session == nil {
		return
	}

	session.Operation = opRenew
	session.Stage = stageAwaitRenewDuration

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("1 день", "renew_duration_1"),
			tgbotapi.NewInlineKeyboardButtonData("7 дней", "renew_duration_7"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("14 дней", "renew_duration_14"),
			tgbotapi.NewInlineKeyboardButtonData("30 дней", "renew_duration_30"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("◀️ Назад", fmt.Sprintf("ad_action_%d", session.Ad.ID)),
		),
	)

	msg := tgbotapi.NewMessage(chatID, "🔄 *Продлить объявление*\n\nВыберите срок продления:")
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard

	sentMsg, err := bot.Send(msg)
	if err == nil {
		addBotMessage(chatID, sentMsg.MessageID)
	}
}

func handleAdRemove(bot *tgbotapi.BotAPI, chatID int64) {
	session := getSession(chatID)
	if session == nil {
		return
	}

	if err := setAdStatus(session.Ad.ID, models.AdStatusInactive); err != nil {
		sendText(bot, chatID, "❌ Не удалось обновить объявление.")
		return
	}

	notifyUser(bot, session.Ad.UserID, fmt.Sprintf("Ваше объявление «%s» снято с биржи. Свяжитесь с %s для повторной публикации.", session.Ad.Title, managerHelpLink))

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("◀️ В меню", "menu_main"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("✅ Объявление #%d снято с биржи.", session.Ad.ID))
	msg.ReplyMarkup = keyboard

	sentMsg, err := bot.Send(msg)
	if err == nil {
		addBotMessage(chatID, sentMsg.MessageID)
		// Удаляем предыдущие сообщения после отправки результата
		session := getSession(chatID)
		if session != nil {
			go scheduleDeletePreviousMessages(bot, chatID, session, sentMsg.MessageID)
		}
	}

	clearSession(chatID)
}

func handleSessionInput(bot *tgbotapi.BotAPI, msg *tgbotapi.Message, session *adSession) {
	session.LastActivity = time.Now()
	text := strings.TrimSpace(msg.Text)

	switch session.Stage {
	case stageAwaitFindAdID:
		handleFindAdIDInput(bot, msg.Chat.ID, text, session)
	case stageAwaitBlacklistAdd:
		handleBlacklistAddInput(bot, msg.Chat.ID, text)
	case stageAwaitBlacklistRemove:
		handleBlacklistRemoveInput(bot, msg.Chat.ID, text)
	case stageAwaitPhoto:
		handlePhotoStage(bot, msg, session)
	case stageAwaitTitle:
		handleTitleInput(bot, msg.Chat.ID, text, session)
	case stageAwaitDescription:
		handleDescriptionInput(bot, msg.Chat.ID, text, session)
	case stageAwaitUsername:
		handleUsernameInput(bot, msg.Chat.ID, text, session)
	case stageAwaitUserId:
		// Ожидаем пересланное сообщение или ввод ID вручную
		// Если это текст с числом, считаем его ID
		if userID, err := strconv.ParseInt(text, 10, 64); err == nil {
			session.Ad.ClientID = text
			session.Ad.UserID = userID
			// Если username не установлен, запрашиваем его отдельно
			if session.Ad.Username == "" {
				log.Printf("ID пользователя введен вручную: UserID=%d, ClientID=%s, Username не установлен", userID, text)
				keyboard := tgbotapi.NewInlineKeyboardMarkup(
					tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonData("⏭ Пропустить (без username)", "skip_username"),
					),
					tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonData("◀️ Назад", getBackCallback(session)),
					),
				)
				msgText := tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("✅ ID пользователя получен: %d\n\n👤 *Введите username для контакта* (например: @username)\n\nИли нажмите \"Пропустить\", если username не нужен.", userID))
				msgText.ParseMode = "Markdown"
				msgText.ReplyMarkup = keyboard
				sentMsg, err := bot.Send(msgText)
				if err == nil {
					addBotMessage(msg.Chat.ID, sentMsg.MessageID)
				}
				session.Stage = stageAwaitUsername
			} else {
				log.Printf("ID пользователя введен вручную: UserID=%d, ClientID=%s, Username=%s", userID, text, session.Ad.Username)
				session.Stage = stageAwaitCategory
				showCategoryPrompt(bot, msg.Chat.ID, session)
			}
		} else {
			sendText(bot, msg.Chat.ID, "❌ Перешлите сообщение от пользователя или введите ID вручную (только цифры).")
		}
	}
}

func handleBlacklistAddInput(bot *tgbotapi.BotAPI, chatID int64, text string) {
	username := normalizeUsername(text)
	if username == "" {
		sendText(bot, chatID, "❌ Введите username в формате @username")
		return
	}

	db.DB.FirstOrCreate(&models.User{}, models.User{Username: username}).Updates(map[string]interface{}{"IsScammer": true})

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("◀️ Назад", "menu_blacklist"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("✅ Добавлен в чёрный список: @%s", username))
	msg.ReplyMarkup = keyboard

	sentMsg, err := bot.Send(msg)
	if err == nil {
		addBotMessage(chatID, sentMsg.MessageID)
		// Удаляем предыдущие сообщения после отправки результата
		session := getSession(chatID)
		if session != nil {
			go scheduleDeletePreviousMessages(bot, chatID, session, sentMsg.MessageID)
		}
	}

	clearSession(chatID)
}

func handleBlacklistRemoveInput(bot *tgbotapi.BotAPI, chatID int64, text string) {
	username := normalizeUsername(text)
	if username == "" {
		sendText(bot, chatID, "❌ Введите username в формате @username")
		return
	}

	result := db.DB.Where("username = ?", username).Updates(&models.User{IsScammer: false})
	if result.Error != nil {
		sendText(bot, chatID, "❌ Ошибка во время обновления чёрного списка.")
		return
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("◀️ Назад", "menu_blacklist"),
		),
	)

	var msgText string
	if result.RowsAffected == 0 {
		msgText = fmt.Sprintf("❌ Пользователь @%s не найден в чёрном списке", username)
	} else {
		msgText = fmt.Sprintf("✅ Удалён из чёрного списка: @%s", username)
	}

	msg := tgbotapi.NewMessage(chatID, msgText)
	msg.ReplyMarkup = keyboard

	sentMsg, err := bot.Send(msg)
	if err == nil {
		addBotMessage(chatID, sentMsg.MessageID)
		// Удаляем предыдущие сообщения после отправки результата
		session := getSession(chatID)
		if session != nil {
			go scheduleDeletePreviousMessages(bot, chatID, session, sentMsg.MessageID)
		}
	}

	clearSession(chatID)
}

func handleSkipPhoto(bot *tgbotapi.BotAPI, chatID int64) {
	session := getSession(chatID)
	if session == nil {
		return
	}

	session.Ad.PhotoID = ""
	session.Ad.PhotoPath = ""
	session.Stage = stageAwaitTitle

	showTitlePrompt(bot, chatID, session)
}

func handlePhotoStage(bot *tgbotapi.BotAPI, msg *tgbotapi.Message, session *adSession) {
	if len(msg.Photo) > 0 {
		photo := msg.Photo[len(msg.Photo)-1]
		file, err := bot.GetFile(tgbotapi.FileConfig{FileID: photo.FileID})
		if err != nil {
			sendText(bot, msg.Chat.ID, "❌ Не удалось сохранить фото, попробуйте ещё раз.")
			return
		}
		session.Ad.PhotoID = photo.FileID
		session.Ad.PhotoPath = file.FilePath
	}

	session.Stage = stageAwaitTitle
	showTitlePrompt(bot, msg.Chat.ID, session)
}

func showTitlePrompt(bot *tgbotapi.BotAPI, chatID int64, session *adSession) {
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("◀️ Назад", getBackCallback(session)),
		),
	)

	text := "📝 *Шаг 2: Заголовок*\n\nВведите заголовок объявления (до 128 символов)."
	if session.Ad.Title != "" {
		text += fmt.Sprintf("\n\nТекущий: %s", session.Ad.Title)
	}

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard

	sentMsg, err := bot.Send(msg)
	if err == nil {
		addBotMessage(chatID, sentMsg.MessageID)
	}
}

func handleTitleInput(bot *tgbotapi.BotAPI, chatID int64, text string, session *adSession) {
	if text == "" {
		sendText(bot, chatID, "❌ Заголовок не может быть пустым.")
		return
	}

	session.Ad.Title = truncate(text, 128)
	session.Stage = stageAwaitDescription

	showDescriptionPrompt(bot, chatID, session)
}

func showDescriptionPrompt(bot *tgbotapi.BotAPI, chatID int64, session *adSession) {
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("◀️ Назад", getBackCallback(session)),
		),
	)

	text := "📄 *Шаг 3: Описание*\n\nВведите описание объявления."
	if session.Ad.Desc != "" {
		text += fmt.Sprintf("\n\nТекущее: %s", truncate(session.Ad.Desc, 100))
	}

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard

	sentMsg, err := bot.Send(msg)
	if err == nil {
		addBotMessage(chatID, sentMsg.MessageID)
	}
}

func handleDescriptionInput(bot *tgbotapi.BotAPI, chatID int64, text string, session *adSession) {
	if text == "" {
		sendText(bot, chatID, "❌ Описание не может быть пустым.")
		return
	}

	session.Ad.Desc = truncate(text, 2048)
	session.Stage = stageAwaitUserId

	// Сразу запрашиваем ID пользователя (переслать сообщение)
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⏭ Пропустить (указать ID вручную)", "skip_user_id"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("◀️ Назад", getBackCallback(session)),
		),
	)

	msg := tgbotapi.NewMessage(chatID, "🆔 *Шаг 4: ID пользователя*\n\nПерешлите любое сообщение от пользователя, чтобы автоматически получить его ID.\n\nИли нажмите \"Пропустить\", чтобы ввести ID вручную.")
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard

	sentMsg, err := bot.Send(msg)
	if err == nil {
		addBotMessage(chatID, sentMsg.MessageID)
	}
}

func showCategoryPrompt(bot *tgbotapi.BotAPI, chatID int64, session *adSession) {
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Услуги", "category_services"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Купля/Продажа", "category_buysell"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Другое", "category_other"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("◀️ Назад", getBackCallback(session)),
		),
	)

	text := "📂 *Шаг 5: Категория*\n\nВыберите категорию объявления."
	if session.Ad.Category != "" {
		text += fmt.Sprintf("\n\nТекущая: %s", categoryLabels[session.Ad.Category])
	}

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard

	sentMsg, err := bot.Send(msg)
	if err == nil {
		addBotMessage(chatID, sentMsg.MessageID)
	}
}

// handleEditSetting обрабатывает нажатие на кнопку редактирования конкретной настройки
func handleEditSetting(bot *tgbotapi.BotAPI, chatID int64, setting string) {
	session := getSession(chatID)
	if session == nil {
		return
	}

	switch setting {
	case "category":
		session.Stage = stageAwaitCategory
		showCategoryPrompt(bot, chatID, session)
	case "mode":
		// Для категории "other" режим не редактируется
		if session.Ad.Category == "other" {
			showAllSettingsPrompt(bot, chatID, session)
			return
		}
		session.Stage = stageAwaitMode
		showModePrompt(bot, chatID, session)
	case "tag":
		session.Stage = stageAwaitTag
		showTagPrompt(bot, chatID, session)
	case "duration":
		session.Stage = stageAwaitDuration
		showDurationPrompt(bot, chatID, session)
	case "premium":
		session.Stage = stageAwaitPremium
		showPremiumPrompt(bot, chatID, session)
	}
}

func handleCategoryCallback(bot *tgbotapi.BotAPI, chatID int64, data string) {
	session := getSession(chatID)
	if session == nil {
		return
	}

	category := strings.TrimPrefix(data, "category_")
	session.Ad.Category = category

	// Если мы редактируем из showAllSettingsPrompt, возвращаемся к нему, иначе продолжаем обычный флоу
	if session.Stage == stageAwaitCategory {
		showAllSettingsPrompt(bot, chatID, session)
	} else {
		// Для категории "other" пропускаем выбор режима и автоматически устанавливаем "general"
		if category == "other" {
			session.Ad.Mode = "general"
			session.Stage = stageAwaitTag
			showTagPrompt(bot, chatID, session)
		} else {
			session.Stage = stageAwaitMode
			showModePrompt(bot, chatID, session)
		}
	}
}

func showModePrompt(bot *tgbotapi.BotAPI, chatID int64, session *adSession) {
	var rows [][]tgbotapi.InlineKeyboardButton
	modeLabelsMap := modeLabels[session.Ad.Category]
	modeValuesMap := modeValues[session.Ad.Category]

	// Итерируемся по modeValuesMap (русское название -> английское значение)
	for label, value := range modeValuesMap {
		// label - русское название (например "Объявление")
		// value - английское значение (например "general")
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(label, fmt.Sprintf("mode_%s", value)),
		))
	}

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("◀️ Назад", getBackCallback(session)),
	))

	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)

	text := "🎯 *Шаг 6: Режим*\n\nВыберите режим объявления."
	if session.Ad.Mode != "" {
		// Ищем русское название по английскому значению
		if modeLabel, ok := modeLabelsMap[session.Ad.Mode]; ok {
			text += fmt.Sprintf("\n\nТекущий: %s", modeLabel)
		}
	}

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard

	sentMsg, err := bot.Send(msg)
	if err == nil {
		addBotMessage(chatID, sentMsg.MessageID)
	}
}

func handleModeCallback(bot *tgbotapi.BotAPI, chatID int64, data string) {
	session := getSession(chatID)
	if session == nil {
		return
	}

	mode := strings.TrimPrefix(data, "mode_")
	session.Ad.Mode = mode

	// Если мы редактируем из showAllSettingsPrompt, возвращаемся к нему, иначе продолжаем обычный флоу
	if session.Stage == stageAwaitMode {
		showAllSettingsPrompt(bot, chatID, session)
	} else {
		session.Stage = stageAwaitTag
		showTagPrompt(bot, chatID, session)
	}
}

func showTagPrompt(bot *tgbotapi.BotAPI, chatID int64, session *adSession) {
	var rows [][]tgbotapi.InlineKeyboardButton
	tagLabelsMap := tagLabels[session.Ad.Category]
	tagValuesMap := tagValues[session.Ad.Category]

	// Разбиваем теги на строки по 2 кнопки
	var currentRow []tgbotapi.InlineKeyboardButton
	// Итерируемся по tagValuesMap (русское название -> английское значение)
	for label, value := range tagValuesMap {
		// label - русское название (например "Дизайнер")
		// value - английское значение (например "designer")
		btn := tgbotapi.NewInlineKeyboardButtonData(label, fmt.Sprintf("tag_%s", value))
		currentRow = append(currentRow, btn)

		if len(currentRow) == 2 {
			rows = append(rows, currentRow)
			currentRow = []tgbotapi.InlineKeyboardButton{}
		}
	}

	if len(currentRow) > 0 {
		rows = append(rows, currentRow)
	}

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("◀️ Назад", getBackCallback(session)),
	))

	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)

	text := "🏷 *Шаг 7: Тег*\n\nВыберите тег объявления."
	if session.Ad.Tag != "" {
		// Ищем русское название по английскому значению
		if tagLabel, ok := tagLabelsMap[session.Ad.Tag]; ok {
			text += fmt.Sprintf("\n\nТекущий: %s", tagLabel)
		}
	}

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard

	sentMsg, err := bot.Send(msg)
	if err == nil {
		addBotMessage(chatID, sentMsg.MessageID)
	}
}

func handleTagCallback(bot *tgbotapi.BotAPI, chatID int64, data string) {
	session := getSession(chatID)
	if session == nil {
		return
	}

	tag := strings.TrimPrefix(data, "tag_")
	session.Ad.Tag = tag

	// Если мы редактируем из showAllSettingsPrompt, возвращаемся к нему, иначе продолжаем обычный флоу
	if session.Stage == stageAwaitTag {
		showAllSettingsPrompt(bot, chatID, session)
	} else {
		session.Stage = stageAwaitDuration
		showDurationPrompt(bot, chatID, session)
	}
}

func showDurationPrompt(bot *tgbotapi.BotAPI, chatID int64, session *adSession) {
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("1 день", "duration_1"),
			tgbotapi.NewInlineKeyboardButtonData("7 дней", "duration_7"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("14 дней", "duration_14"),
			tgbotapi.NewInlineKeyboardButtonData("30 дней", "duration_30"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("◀️ Назад", getBackCallback(session)),
		),
	)

	text := "⏱ *Шаг 8: Срок действия*\n\nВыберите срок отображения объявления."

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard

	sentMsg, err := bot.Send(msg)
	if err == nil {
		addBotMessage(chatID, sentMsg.MessageID)
	}
}

func handleDurationCallback(bot *tgbotapi.BotAPI, chatID int64, data string) {
	session := getSession(chatID)
	if session == nil {
		return
	}

	daysStr := strings.TrimPrefix(data, "duration_")
	days, err := strconv.Atoi(daysStr)
	if err != nil || !isValidDuration(days) {
		sendText(bot, chatID, "❌ Неверный срок.")
		return
	}

	session.DurationDays = days

	// Если мы редактируем из showAllSettingsPrompt, возвращаемся к нему, иначе продолжаем обычный флоу
	if session.Stage == stageAwaitDuration {
		showAllSettingsPrompt(bot, chatID, session)
	} else {
		session.Stage = stageAwaitPremium
		showPremiumPrompt(bot, chatID, session)
	}
}

func showPremiumPrompt(bot *tgbotapi.BotAPI, chatID int64, session *adSession) {
	var exclude *uint
	if session.Operation != opCreate {
		exclude = &session.Ad.ID
	}
	count, err := activePremiumCount(exclude)
	if err != nil {
		sendText(bot, chatID, "❌ Не удалось проверить лимит премиум-объявлений.")
		return
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Да", "premium_yes"),
			tgbotapi.NewInlineKeyboardButtonData("❌ Нет", "premium_no"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("◀️ Назад", getBackCallback(session)),
		),
	)

	text := "⭐ *Шаг 9: Премиум размещение*\n\nПремиум объявление будет отображаться вверху списка."
	if count >= 3 {
		text += "\n\n⚠️ Лимит премиум-объявлений (3) исчерпан. Сначала снимите одно из текущих."
	}

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard

	sentMsg, err := bot.Send(msg)
	if err == nil {
		addBotMessage(chatID, sentMsg.MessageID)
	}
}

func handlePremiumCallback(bot *tgbotapi.BotAPI, chatID int64, data string) {
	session := getSession(chatID)
	if session == nil {
		return
	}

	if data == "premium_yes" {
		var exclude *uint
		if session.Operation != opCreate {
			exclude = &session.Ad.ID
		}
		count, err := activePremiumCount(exclude)
		if err != nil {
			sendText(bot, chatID, "❌ Не удалось проверить лимит премиум-объявлений.")
			return
		}
		if count >= 3 {
			sendText(bot, chatID, "⚠️ Лимит премиум-объявлений (3) исчерпан. Сначала снимите одно из текущих.")
			return
		}
		session.Ad.IsPremium = true
	} else {
		session.Ad.IsPremium = false
	}

	// После выбора премиума показываем предпросмотр (если ClientID уже установлен) или продолжаем
	if session.Ad.ClientID != "" {
		session.Stage = stageAwaitConfirmation
		showConfirmationPrompt(bot, chatID, session)
	} else if session.Stage == stageAwaitPremium {
		showAllSettingsPrompt(bot, chatID, session)
	} else {
		// Если ClientID не установлен, должны были получить его раньше - возвращаемся к настройкам
		showAllSettingsPrompt(bot, chatID, session)
	}
}

// handleSkipUserID позволяет пропустить автоматическое получение ID и ввести его вручную
func handleSkipUserID(bot *tgbotapi.BotAPI, chatID int64) {
	session := getSession(chatID)
	if session == nil {
		return
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("◀️ Назад", getBackCallback(session)),
		),
	)

	msg := tgbotapi.NewMessage(chatID, "🆔 *Ввод ID клиента*\n\nВведите ID клиента вручную (только цифры):")
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard

	sentMsg, err := bot.Send(msg)
	if err == nil {
		addBotMessage(chatID, sentMsg.MessageID)
	}
	session.Stage = stageAwaitUserId
}

func handleSkipUsername(bot *tgbotapi.BotAPI, chatID int64) {
	session := getSession(chatID)
	if session == nil {
		return
	}

	// Пропускаем username, оставляем пустым (не используем user_{id})
	session.Ad.Username = ""
	log.Printf("Username пропущен, оставляем пустым")
	session.Stage = stageAwaitCategory
	showCategoryPrompt(bot, chatID, session)
}

func handleUsernameInput(bot *tgbotapi.BotAPI, chatID int64, text string, session *adSession) {
	username := normalizeUsername(text)
	if username == "" {
		sendText(bot, chatID, "❌ Введите username в формате @username или нажмите \"Пропустить\".")
		return
	}

	session.Ad.Username = username
	log.Printf("Username введен вручную: %s", username)
	session.Stage = stageAwaitCategory
	showCategoryPrompt(bot, chatID, session)
}

func handleFindAdIDInput(bot *tgbotapi.BotAPI, chatID int64, text string, session *adSession) {
	text = strings.TrimSpace(text)
	if text == "" {
		sendText(bot, chatID, "❌ ID клиента не может быть пустым. Введите ID или перешлите сообщение от пользователя.")
		return
	}

	// Проверяем, что ID клиента содержит только цифры
	clientID, err := strconv.ParseInt(text, 10, 64)
	if err != nil {
		sendText(bot, chatID, "❌ ID клиента должен быть числом. Введите ID или перешлите сообщение от пользователя.")
		return
	}

	clientIDStr := strconv.FormatInt(clientID, 10)
	log.Printf("Поиск объявлений для ClientID: %s", clientIDStr)

	// Ищем все объявления по ClientID
	var ads []models.Ad
	if err := db.DB.Where("client_id = ?", clientIDStr).Order("created_at DESC").Find(&ads).Error; err != nil {
		log.Printf("Ошибка поиска объявлений: %v", err)
		sendText(bot, chatID, "❌ Ошибка при поиске объявлений.")
		return
	}

	log.Printf("Найдено объявлений: %d", len(ads))

	if len(ads) == 0 {
		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("◀️ Назад", "menu_main"),
			),
		)
		msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("❌ Объявления для клиента ID %s не найдены.", clientIDStr))
		msg.ReplyMarkup = keyboard
		sentMsg, err := bot.Send(msg)
		if err == nil {
			addBotMessage(chatID, sentMsg.MessageID)
			// Удаляем предыдущие сообщения после отправки результата
			go scheduleDeletePreviousMessages(bot, chatID, session, sentMsg.MessageID)
		} else {
			log.Printf("Ошибка отправки сообщения: %v", err)
		}
		clearSession(chatID)
		return
	}

	// Показываем результаты
	log.Printf("Вызов handleFindAdResults для %d объявлений", len(ads))
	handleFindAdResults(bot, chatID, ads, session)
}

// handleFindAdResults показывает результаты поиска объявлений
func handleFindAdResults(bot *tgbotapi.BotAPI, chatID int64, ads []models.Ad, session *adSession) {
	// Если одно объявление - показываем его
	if len(ads) == 1 {
		session.Ad = ads[0]
		session.Stage = stageAwaitAction
		showAdDetailsWithActions(bot, chatID, ads[0])
		return
	}

	// Если несколько объявлений - показываем список
	session.Stage = stageAwaitSelectAd
	var textBuilder strings.Builder
	textBuilder.WriteString(fmt.Sprintf("📋 *Найдено объявлений: %d*\n\n", len(ads)))

	var rows [][]tgbotapi.InlineKeyboardButton
	for i, ad := range ads {
		if i >= 10 { // Ограничение на количество кнопок
			textBuilder.WriteString(fmt.Sprintf("\n... и ещё %d объявлений", len(ads)-10))
			break
		}
		var status string
		switch ad.Status {
		case models.AdStatusExpired:
			status = "🔴 Истекло"
		case models.AdStatusInactive:
			status = "⚫ Снято"
		default:
			status = "🟢 Активно"
		}
		textBuilder.WriteString(fmt.Sprintf("%d. %s - %s\n", ad.ID, ad.Title, status))
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				fmt.Sprintf("#%d: %s", ad.ID, truncate(ad.Title, 30)),
				fmt.Sprintf("select_ad_%d", ad.ID),
			),
		))
	}

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("◀️ Назад", "menu_main"),
	))

	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)
	msg := tgbotapi.NewMessage(chatID, textBuilder.String())
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard

	log.Printf("Отправка результатов поиска: найдено %d объявлений", len(ads))
	sentMsg, err := bot.Send(msg)
	if err != nil {
		log.Printf("Ошибка отправки результатов поиска: %v", err)
		sendText(bot, chatID, "❌ Не удалось отправить результаты поиска.")
		return
	}

	addBotMessage(chatID, sentMsg.MessageID)
	// Удаляем предыдущие сообщения после отправки списка объявлений
	if session != nil {
		go scheduleDeletePreviousMessages(bot, chatID, session, sentMsg.MessageID)
	}
}

func handleSelectAd(bot *tgbotapi.BotAPI, chatID int64, data string) {
	adIDStr := strings.TrimPrefix(data, "select_ad_")
	adID, err := strconv.ParseUint(adIDStr, 10, 32)
	if err != nil {
		sendText(bot, chatID, "❌ Неверный ID объявления.")
		return
	}

	var ad models.Ad
	if err := db.DB.First(&ad, uint(adID)).Error; err != nil {
		sendText(bot, chatID, "❌ Объявление не найдено.")
		return
	}

	session := getSession(chatID)
	if session == nil {
		// Создаём сессию, если её нет
		session = &adSession{
			Stage:         stageAwaitAction,
			LastActivity:  time.Now(),
			ChatID:        chatID,
			BotMessageIDs: []int{},
			Ad:            ad,
		}
		setSession(chatID, session)
	} else {
		session.Ad = ad
		session.Stage = stageAwaitAction
		session.LastActivity = time.Now()
	}

	showAdDetailsWithActions(bot, chatID, ad)
}

func showAdDetailsWithActions(bot *tgbotapi.BotAPI, chatID int64, ad models.Ad) {
	text := renderAdSummaryWithExpiry(ad)

	var rows [][]tgbotapi.InlineKeyboardButton

	// Если объявление не выложено (статус inactive или неактивно)
	if ad.Status == models.AdStatusInactive || ad.ExpiresAt.Before(time.Now()) {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Выложить", "ad_publish"),
		))
	}

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("✏️ Изменить", "ad_edit"),
	))

	if ad.Status == models.AdStatusActive {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔄 Продлить", "ad_renew"),
			tgbotapi.NewInlineKeyboardButtonData("❌ Снять", "ad_remove"),
		))
	}

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("◀️ Назад", "menu_main"),
	))

	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard

	sentMsg, err := bot.Send(msg)
	if err == nil {
		addBotMessage(chatID, sentMsg.MessageID)
		// Удаляем предыдущие сообщения после отправки деталей объявления
		session := getSession(chatID)
		if session != nil {
			go scheduleDeletePreviousMessages(bot, chatID, session, sentMsg.MessageID)
		}
	}
}

func handleAdPublish(bot *tgbotapi.BotAPI, chatID int64) {
	session := getSession(chatID)
	if session == nil {
		return
	}

	// Активируем объявление
	session.Ad.Status = models.AdStatusActive
	session.Ad.PreExpiryNotified = false
	if session.Ad.ExpiresAt.Before(time.Now()) {
		// Если срок истёк, устанавливаем новый срок (7 дней по умолчанию)
		session.Ad.ExpiresAt = time.Now().Add(7 * 24 * time.Hour)
	}

	if err := db.DB.Save(&session.Ad).Error; err != nil {
		sendText(bot, chatID, "❌ Не удалось выложить объявление.")
		return
	}

	notifyUser(bot, session.Ad.UserID, fmt.Sprintf("Ваше объявление «%s» выложено на биржу. Свяжитесь с %s для управления.", session.Ad.Title, managerHelpLink))

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("◀️ В меню", "menu_main"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("✅ Объявление #%d выложено на биржу.", session.Ad.ID))
	msg.ReplyMarkup = keyboard

	sentMsg, err := bot.Send(msg)
	if err == nil {
		addBotMessage(chatID, sentMsg.MessageID)
		// Удаляем предыдущие сообщения после отправки результата
		session := getSession(chatID)
		if session != nil {
			go scheduleDeletePreviousMessages(bot, chatID, session, sentMsg.MessageID)
		}
	}

	clearSession(chatID)
}

func showConfirmationPrompt(bot *tgbotapi.BotAPI, chatID int64, session *adSession) int {
	preview := renderAdPreview(session)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Подтвердить", "confirm_yes"),
			tgbotapi.NewInlineKeyboardButtonData("✏️ Изменить", "edit_after_preview"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("◀️ Назад", getBackCallback(session)),
		),
	)

	msg := tgbotapi.NewMessage(chatID, preview)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard

	sentMsg, err := bot.Send(msg)
	if err != nil {
		log.Printf("Ошибка отправки предпросмотра: %v", err)
		return 0
	}

	addBotMessage(chatID, sentMsg.MessageID)

	// Удаляем предыдущие сообщения после отправки предпросмотра (кроме самого предпросмотра)
	go scheduleDeletePreviousMessages(bot, chatID, session, sentMsg.MessageID)

	return sentMsg.MessageID
}

func handleEditAfterPreview(bot *tgbotapi.BotAPI, chatID int64) {
	session := getSession(chatID)
	if session == nil {
		return
	}

	// Показываем все настройки сразу для удобного редактирования
	showAllSettingsPrompt(bot, chatID, session)
}

// showAllSettingsPrompt показывает все настройки объявления сразу с кнопками для редактирования
func showAllSettingsPrompt(bot *tgbotapi.BotAPI, chatID int64, session *adSession) {
	var text strings.Builder
	text.WriteString("⚙️ *Настройки объявления*\n\n")

	// Категория
	categoryLabel := categoryLabels[session.Ad.Category]
	if categoryLabel == "" {
		categoryLabel = session.Ad.Category
	}
	text.WriteString(fmt.Sprintf("📂 Категория: %s\n", categoryLabel))

	// Режим (для категории "other" не показываем, так как он автоматический)
	if session.Ad.Category != "other" {
		modeLabel := modeLabels[session.Ad.Category][session.Ad.Mode]
		if modeLabel == "" {
			modeLabel = session.Ad.Mode
		}
		text.WriteString(fmt.Sprintf("🎯 Режим: %s\n", modeLabel))
	}

	// Тег
	tagLabel := tagLabels[session.Ad.Category][session.Ad.Tag]
	if tagLabel == "" {
		tagLabel = session.Ad.Tag
	}
	text.WriteString(fmt.Sprintf("🏷 Тег: %s\n", tagLabel))

	// Премиум
	premiumLabel := "нет"
	if session.Ad.IsPremium {
		premiumLabel = "да"
	}
	text.WriteString(fmt.Sprintf("⭐ Премиум: %s\n", premiumLabel))

	// Срок действия
	durationLabel := "не задан"
	if session.DurationDays > 0 {
		durationLabel = fmt.Sprintf("%d дн.", session.DurationDays)
	} else if !session.Ad.ExpiresAt.IsZero() {
		durationLabel = session.Ad.ExpiresAt.Format("02.01.2006")
	}
	text.WriteString(fmt.Sprintf("⏱ Срок действия: %s\n\n", durationLabel))

	text.WriteString("Выберите, что хотите изменить:")

	var rows [][]tgbotapi.InlineKeyboardButton

	// Кнопки для редактирования настроек
	// Для категории "other" не показываем кнопку редактирования режима
	if session.Ad.Category != "other" {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📂 Категория", "category_edit"),
			tgbotapi.NewInlineKeyboardButtonData("🎯 Режим", "mode_edit"),
		))
	} else {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📂 Категория", "category_edit"),
		))
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🏷 Тег", "tag_edit"),
		tgbotapi.NewInlineKeyboardButtonData("⏱ Срок", "duration_edit"),
	))
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("⭐ Премиум", "premium_edit"),
	))
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("✅ Сохранить", "save_from_settings"),
	))
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("◀️ Назад", "back"),
	))

	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)

	msg := tgbotapi.NewMessage(chatID, text.String())
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard

	sentMsg, err := bot.Send(msg)
	if err == nil {
		addBotMessage(chatID, sentMsg.MessageID)
		// Удаляем предыдущие сообщения после отправки настроек
		go scheduleDeletePreviousMessages(bot, chatID, session, sentMsg.MessageID)
	}
}

// handleSaveFromSettings сохраняет объявление из экрана настроек
func handleSaveFromSettings(bot *tgbotapi.BotAPI, chatID int64) {
	session := getSession(chatID)
	if session == nil {
		return
	}

	// Проверяем, что ClientID установлен
	if session.Ad.ClientID == "" {
		sendText(bot, chatID, "❌ Необходимо указать ID клиента. Вернитесь к предпросмотру и введите ID клиента.")
		return
	}

	// Если UserID не установлен, устанавливаем его из ClientID
	if session.Ad.UserID == 0 && session.Ad.ClientID != "" {
		if userID, err := strconv.ParseInt(session.Ad.ClientID, 10, 64); err == nil {
			session.Ad.UserID = userID
		}
	}

	// Сохраняем объявление
	if err := persistAd(bot, session); err != nil {
		sendText(bot, chatID, "❌ Не удалось сохранить объявление: "+err.Error())
		return
	}

	deleteBotMessages(bot, chatID, session)
	clearSession(chatID)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("◀️ В меню", "menu_main"),
		),
	)

	var text string
	if session.Operation == opCreate {
		text = fmt.Sprintf("✅ Объявление #%d опубликовано.", session.Ad.ID)
	} else {
		text = fmt.Sprintf("✅ Объявление #%d обновлено.", session.Ad.ID)
	}

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = keyboard

	sentMsg, err := bot.Send(msg)
	if err == nil {
		addBotMessage(chatID, sentMsg.MessageID)
	}
}

func handleConfirmYes(bot *tgbotapi.BotAPI, chatID int64) {
	session := getSession(chatID)
	if session == nil {
		return
	}

	// Если UserID не установлен, устанавливаем его из ClientID
	if session.Ad.UserID == 0 && session.Ad.ClientID != "" {
		if userID, err := strconv.ParseInt(session.Ad.ClientID, 10, 64); err == nil {
			session.Ad.UserID = userID
		}
	}

	if err := persistAd(bot, session); err != nil {
		sendText(bot, chatID, "❌ Не удалось сохранить объявление: "+err.Error())
		return
	}

	deleteBotMessages(bot, chatID, session)
	clearSession(chatID)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("◀️ В меню", "menu_main"),
		),
	)

	var text string
	if session.Operation == opCreate {
		text = fmt.Sprintf("✅ Объявление #%d опубликовано.", session.Ad.ID)
	} else {
		text = fmt.Sprintf("✅ Объявление #%d обновлено.", session.Ad.ID)
	}

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = keyboard

	sentMsg, err := bot.Send(msg)
	if err == nil {
		addBotMessage(chatID, sentMsg.MessageID)
	}
}

func handleConfirmNo(bot *tgbotapi.BotAPI, chatID int64) {
	session := getSession(chatID)
	if session != nil {
		deleteBotMessages(bot, session.ChatID, session)
	}
	clearSession(chatID)
	showMainMenu(bot, chatID)
}

func handleRenewDurationCallback(bot *tgbotapi.BotAPI, chatID int64, data string) {
	session := getSession(chatID)
	if session == nil {
		return
	}

	daysStr := strings.TrimPrefix(data, "renew_duration_")
	days, err := strconv.Atoi(daysStr)
	if err != nil || !isValidDuration(days) {
		sendText(bot, chatID, "❌ Неверный срок.")
		return
	}

	session.Ad.Status = models.AdStatusActive
	session.Ad.PreExpiryNotified = false
	session.Ad.ExpiresAt = time.Now().Add(time.Duration(days) * 24 * time.Hour)
	if err := db.DB.Save(&session.Ad).Error; err != nil {
		sendText(bot, chatID, "❌ Не удалось обновить объявление.")
		return
	}

	notifyUser(bot, session.Ad.UserID, fmt.Sprintf("Ваше объявление «%s» продлено до %s.", session.Ad.Title, session.Ad.ExpiresAt.Format("02.01.2006")))

	deleteBotMessages(bot, chatID, session)
	clearSession(chatID)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("◀️ В меню", "menu_main"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("✅ Объявление #%d продлено до %s.", session.Ad.ID, session.Ad.ExpiresAt.Format("02.01.2006 15:04")))
	msg.ReplyMarkup = keyboard

	sentMsg, err := bot.Send(msg)
	if err == nil {
		addBotMessage(chatID, sentMsg.MessageID)
	}
}

func handleBack(bot *tgbotapi.BotAPI, chatID int64) {
	session := getSession(chatID)
	if session == nil {
		showMainMenu(bot, chatID)
		return
	}

	// Возвращаемся к предыдущему этапу
	switch session.Stage {
	case stageAwaitTitle:
		session.Stage = stageAwaitPhoto
		showPhotoPrompt(bot, chatID, session)
	case stageAwaitDescription:
		session.Stage = stageAwaitTitle
		showTitlePrompt(bot, chatID, session)
	case stageAwaitUsername:
		session.Stage = stageAwaitUserId
		// Показываем запрос ID
		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("⏭ Пропустить (указать ID вручную)", "skip_user_id"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("◀️ Назад", getBackCallback(session)),
			),
		)
		msg := tgbotapi.NewMessage(chatID, "🆔 *ID пользователя*\n\nПерешлите любое сообщение от пользователя, чтобы автоматически получить его ID.\n\nИли нажмите \"Пропустить\", чтобы ввести ID вручную.")
		msg.ParseMode = "Markdown"
		msg.ReplyMarkup = keyboard
		sentMsg, err := bot.Send(msg)
		if err == nil {
			addBotMessage(chatID, sentMsg.MessageID)
		}
	case stageAwaitCategory:
		session.Stage = stageAwaitUserId
		// Показываем запрос ID вместо username
		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("⏭ Пропустить (указать ID вручную)", "skip_user_id"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("◀️ Назад", getBackCallback(session)),
			),
		)
		msg := tgbotapi.NewMessage(chatID, "🆔 *ID пользователя*\n\nПерешлите любое сообщение от пользователя, чтобы автоматически получить его ID.\n\nИли нажмите \"Пропустить\", чтобы ввести ID вручную.")
		msg.ParseMode = "Markdown"
		msg.ReplyMarkup = keyboard
		sentMsg, err := bot.Send(msg)
		if err == nil {
			addBotMessage(chatID, sentMsg.MessageID)
		}
	case stageAwaitMode:
		session.Stage = stageAwaitCategory
		showCategoryPrompt(bot, chatID, session)
	case stageAwaitTag:
		// Для категории "other" пропускаем режим и возвращаемся к категории
		if session.Ad.Category == "other" {
			session.Stage = stageAwaitCategory
			showCategoryPrompt(bot, chatID, session)
		} else {
			session.Stage = stageAwaitMode
			showModePrompt(bot, chatID, session)
		}
	case stageAwaitDuration:
		session.Stage = stageAwaitTag
		showTagPrompt(bot, chatID, session)
	case stageAwaitPremium:
		session.Stage = stageAwaitDuration
		showDurationPrompt(bot, chatID, session)
	case stageAwaitUserId:
		session.Stage = stageAwaitDescription
		showDescriptionPrompt(bot, chatID, session)
	case stageAwaitConfirmation:
		session.Stage = stageAwaitPremium
		showPremiumPrompt(bot, chatID, session)
	}
}

func showPhotoPrompt(bot *tgbotapi.BotAPI, chatID int64, session *adSession) {
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⏭ Пропустить", "skip_photo"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("◀️ Назад", getBackCallback(session)),
		),
	)

	msg := tgbotapi.NewMessage(chatID, "📸 *Шаг 1: Фото*\n\nОтправьте фото объявления или пропустите этот шаг.")
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard

	sentMsg, err := bot.Send(msg)
	if err == nil {
		addBotMessage(chatID, sentMsg.MessageID)
	}
}

func getBackCallback(session *adSession) string {
	// Определяем, куда вернуться при нажатии "Назад"
	if session.Operation == opEdit && session.Stage == stageAwaitPhoto {
		return fmt.Sprintf("ad_action_%d", session.Ad.ID)
	}
	return "back"
}

func persistAd(bot *tgbotapi.BotAPI, session *adSession) error {
	// Валидация обязательных полей
	if session.Ad.Title == "" {
		return fmt.Errorf("заголовок не может быть пустым")
	}
	if session.Ad.Desc == "" {
		return fmt.Errorf("описание не может быть пустым")
	}
	// Username опционален - если не указан, оставляем пустым (не используем user_{id})
	// Это нормально, так как для поиска в профиле используется client_id, а не username
	if session.Ad.Username == "" {
		log.Printf("Предупреждение: Username не указан, оставляем пустым. ClientID=%s", session.Ad.ClientID)
	}
	if session.Ad.Category == "" {
		return fmt.Errorf("категория не может быть пустой")
	}
	// Для категории "other" режим автоматически устанавливается как "general"
	// Также исправляем, если случайно сохранилось русское название "Объявление"
	if session.Ad.Category == "other" {
		if session.Ad.Mode == "" || session.Ad.Mode == "Объявление" {
			session.Ad.Mode = "general"
		}
	}
	if session.Ad.Mode == "" {
		return fmt.Errorf("режим не может быть пустым")
	}
	if session.Ad.Tag == "" {
		return fmt.Errorf("тег не может быть пустым")
	}
	if session.Ad.ClientID == "" {
		return fmt.Errorf("ID клиента не может быть пустым")
	}
	if session.DurationDays == 0 && session.Operation == opCreate {
		return fmt.Errorf("срок действия не может быть пустым")
	}

	// Если UserID не установлен, устанавливаем его из ClientID
	if session.Ad.UserID == 0 && session.Ad.ClientID != "" {
		if userID, err := strconv.ParseInt(session.Ad.ClientID, 10, 64); err == nil {
			session.Ad.UserID = userID
		} else {
			log.Printf("Предупреждение: не удалось преобразовать ClientID %s в UserID: %v", session.Ad.ClientID, err)
		}
	}

	now := time.Now()
	if session.DurationDays > 0 {
		session.Ad.ExpiresAt = now.Add(time.Duration(session.DurationDays) * 24 * time.Hour)
	} else if session.Ad.ExpiresAt.IsZero() && session.Operation == opCreate {
		// Если срок не установлен, устанавливаем по умолчанию 7 дней
		session.Ad.ExpiresAt = now.Add(7 * 24 * time.Hour)
	}

	session.Ad.PreExpiryNotified = false
	session.Ad.Status = models.AdStatusActive

	log.Printf("Сохранение объявления: Title=%s, Username=%s, ClientID=%s, UserID=%d, Category=%s, Mode=%s, Tag=%s",
		session.Ad.Title, session.Ad.Username, session.Ad.ClientID, session.Ad.UserID, session.Ad.Category, session.Ad.Mode, session.Ad.Tag)

	switch session.Operation {
	case opCreate:
		if err := db.DB.Create(&session.Ad).Error; err != nil {
			log.Printf("Ошибка создания объявления: %v", err)
			return err
		}
		log.Printf("Объявление создано: ID=%d, Username=%s, ClientID=%s, UserID=%d", session.Ad.ID, session.Ad.Username, session.Ad.ClientID, session.Ad.UserID)
	case opEdit:
		if err := db.DB.Save(&session.Ad).Error; err != nil {
			log.Printf("Ошибка обновления объявления: %v", err)
			return err
		}
		log.Printf("Объявление обновлено: ID=%d, Username=%s, ClientID=%s, UserID=%d", session.Ad.ID, session.Ad.Username, session.Ad.ClientID, session.Ad.UserID)
	}

	// Уведомляем пользователя о публикации объявления
	if session.Ad.UserID != 0 {
		message := fmt.Sprintf("✅ Ваше объявление «%s» опубликовано до %s.\n\nДля управления обратитесь к %s.", session.Ad.Title, session.Ad.ExpiresAt.Format("02.01.2006"), managerHelpLink)
		notifyUser(bot, session.Ad.UserID, message)
	} else {
		log.Printf("Предупреждение: UserID равен 0, уведомление не отправлено. ClientID=%s", session.Ad.ClientID)
	}

	return nil
}

func setAdStatus(adID uint, status string) error {
	return db.DB.Model(&models.Ad{}).Where("id = ?", adID).Updates(map[string]interface{}{
		"status":              status,
		"pre_expiry_notified": false,
	}).Error
}

func notifyUser(bot *tgbotapi.BotAPI, chatID int64, message string) {
	if chatID == 0 || strings.TrimSpace(message) == "" {
		return
	}
	msg := tgbotapi.NewMessage(chatID, message)
	if _, err := bot.Send(msg); err != nil {
		log.Printf("failed to notify user %d: %v", chatID, err)
	}
}

func persistSessionsCleanup() {
	sessionRegistry.Lock()
	defer sessionRegistry.Unlock()
	for chatID, session := range sessionRegistry.data {
		if time.Since(session.LastActivity) > sessionTimeoutDuration {
			delete(sessionRegistry.data, chatID)
		}
	}
}

// deleteMessageWithEffect удаляет сообщение с эффектом "таноса" (редактирование перед удалением)
func deleteMessageWithEffect(bot *tgbotapi.BotAPI, chatID int64, messageID int) {
	// Сначала редактируем сообщение для эффекта "таноса" (постепенное исчезновение)
	// Эффект "таноса" - постепенное уменьшение текста до точек
	editMsg1 := tgbotapi.NewEditMessageText(chatID, messageID, ".")
	bot.Request(editMsg1)
	time.Sleep(200 * time.Millisecond)

	editMsg2 := tgbotapi.NewEditMessageText(chatID, messageID, "..")
	bot.Request(editMsg2)
	time.Sleep(200 * time.Millisecond)

	editMsg3 := tgbotapi.NewEditMessageText(chatID, messageID, "...")
	bot.Request(editMsg3)
	time.Sleep(200 * time.Millisecond)

	// Удаляем сообщение
	time.Sleep(100 * time.Millisecond)
	_, _ = bot.Request(tgbotapi.NewDeleteMessage(chatID, messageID))
}

// deleteMessage удаляет сообщение без эффекта (для сообщений менеджера)
func deleteMessage(bot *tgbotapi.BotAPI, chatID int64, messageID int) {
	time.Sleep(4 * time.Second) // Увеличена задержка для плавного удаления сообщений менеджера
	_, _ = bot.Request(tgbotapi.NewDeleteMessage(chatID, messageID))
}

// deleteBotMessagesWithEffect удаляет все сообщения бота с эффектом "таноса" постепенно
func deleteBotMessagesWithEffect(bot *tgbotapi.BotAPI, chatID int64, session *adSession) {
	if session == nil || len(session.BotMessageIDs) == 0 {
		return
	}

	// Удаляем сообщения постепенно с задержкой между удалениями
	for i, msgID := range session.BotMessageIDs {
		delay := time.Duration(i) * 150 * time.Millisecond // Задержка между удалениями
		go func(id int, delayTime time.Duration) {
			time.Sleep(delayTime)
			deleteMessageWithEffect(bot, chatID, id)
		}(msgID, delay)
	}

	// Очищаем список сообщений
	session.BotMessageIDs = []int{}
}

// deleteBotMessages удаляет все сообщения бота (старая функция для обратной совместимости)
func deleteBotMessages(bot *tgbotapi.BotAPI, chatID int64, session *adSession) {
	deleteBotMessagesWithEffect(bot, chatID, session)
}

// scheduleDeletePreviousMessages планирует удаление предыдущих сообщений с задержкой
func scheduleDeletePreviousMessages(bot *tgbotapi.BotAPI, chatID int64, session *adSession, keepMsgID int) {
	// Ждем немного, чтобы новое сообщение успело отобразиться
	time.Sleep(800 * time.Millisecond)

	// Удаляем все предыдущие сообщения
	if session == nil {
		return
	}

	// Получаем актуальную сессию (она могла измениться)
	sessionRegistry.Lock()
	currentSession := sessionRegistry.data[chatID]
	sessionRegistry.Unlock()

	if currentSession == nil || len(currentSession.BotMessageIDs) == 0 {
		return
	}

	var toDelete []int
	var toKeep []int

	// Разделяем сообщения на те, которые нужно удалить, и те, которые нужно оставить
	for _, msgID := range currentSession.BotMessageIDs {
		if keepMsgID > 0 && msgID == keepMsgID {
			toKeep = append(toKeep, msgID)
		} else {
			toDelete = append(toDelete, msgID)
		}
	}

	// Удаляем сообщения постепенно с эффектом "таноса"
	for i, msgID := range toDelete {
		delay := time.Duration(i) * 200 * time.Millisecond // Задержка между удалениями
		go func(id int, delayTime time.Duration) {
			time.Sleep(delayTime)
			deleteMessageWithEffect(bot, chatID, id)
		}(msgID, delay)
	}

	// Обновляем список сообщений
	sessionRegistry.Lock()
	updatedSession := sessionRegistry.data[chatID]
	if updatedSession != nil {
		if keepMsgID > 0 {
			// Оставляем только указанное сообщение
			updatedSession.BotMessageIDs = toKeep
		} else if len(toKeep) > 0 {
			// Если есть сообщения для сохранения, оставляем их
			updatedSession.BotMessageIDs = toKeep
		} else if len(currentSession.BotMessageIDs) > 0 {
			// Оставляем только последнее сообщение (предполагаем, что оно новое)
			lastMsgID := currentSession.BotMessageIDs[len(currentSession.BotMessageIDs)-1]
			// Проверяем, что последнее сообщение не в списке для удаления
			shouldKeepLast := true
			for _, id := range toDelete {
				if id == lastMsgID {
					shouldKeepLast = false
					break
				}
			}
			if shouldKeepLast {
				updatedSession.BotMessageIDs = []int{lastMsgID}
			} else {
				updatedSession.BotMessageIDs = []int{}
			}
		}
	}
	sessionRegistry.Unlock()
}

func addBotMessage(chatID int64, messageID int) {
	session := getSession(chatID)
	if session != nil {
		sessionRegistry.Lock()
		session.BotMessageIDs = append(session.BotMessageIDs, messageID)
		sessionRegistry.Unlock()
	}
}

func setSession(chatID int64, session *adSession) {
	sessionRegistry.Lock()
	session.ChatID = chatID
	sessionRegistry.data[chatID] = session
	sessionRegistry.Unlock()
}

func getSession(chatID int64) *adSession {
	sessionRegistry.Lock()
	defer sessionRegistry.Unlock()
	return sessionRegistry.data[chatID]
}

func clearSession(chatID int64) {
	sessionRegistry.Lock()
	delete(sessionRegistry.data, chatID)
	sessionRegistry.Unlock()
}

func isCommand(text, cmd string) bool {
	return strings.HasPrefix(strings.ToLower(text), strings.ToLower(cmd))
}

func truncate(s string, limit int) string {
	// Преобразуем строку в руны для правильной обработки UTF-8
	runes := []rune(s)
	if len(runes) <= limit {
		return s
	}
	// Обрезаем по рунам, а не по байтам, чтобы не обрезать UTF-8 последовательность
	return string(runes[:limit])
}

func normalizeUsername(value string) string {
	return strings.Trim(strings.TrimSpace(strings.TrimPrefix(value, "@")), "@ ")
}

func isValidDuration(days int) bool {
	switch days {
	case 1, 7, 14, 30:
		return true
	default:
		return false
	}
}

func renderAdSummaryWithExpiry(ad models.Ad) string {
	premium := "нет"
	if ad.IsPremium {
		premium = "да"
	}

	categoryLabel := categoryLabels[ad.Category]
	if categoryLabel == "" {
		categoryLabel = ad.Category
	}

	modeLabel := modeLabels[ad.Category][ad.Mode]
	if modeLabel == "" {
		modeLabel = ad.Mode
	}

	tagLabel := tagLabels[ad.Category][ad.Tag]
	if tagLabel == "" {
		tagLabel = ad.Tag
	}

	var statusLabel string
	switch ad.Status {
	case models.AdStatusExpired:
		statusLabel = "Истекло"
	case models.AdStatusInactive:
		statusLabel = "Снято"
	default:
		statusLabel = "Активно"
	}

	// Экранируем специальные символы Markdown в описании
	escapedDesc := escapeMarkdown(ad.Desc)
	// Telegram имеет лимит 4096 символов на сообщение, оставляем запас для остального текста
	maxDescLength := 3500
	if len(escapedDesc) > maxDescLength {
		escapedDesc = escapedDesc[:maxDescLength] + "..."
	}

	text := fmt.Sprintf(
		"📋 *Объявление #%d*\n\n"+
			"📝 Заголовок: %s\n"+
			"📄 Описание: %s\n"+
			"👤 Контакт: @%s\n"+
			"📂 Категория: %s\n"+
			"🎯 Режим: %s\n"+
			"🏷 Тег: %s\n"+
			"⭐ Премиум: %s\n"+
			"📊 Статус: %s",
		ad.ID,
		escapeMarkdown(ad.Title),
		escapedDesc,
		escapeMarkdown(ad.Username),
		categoryLabel,
		modeLabel,
		tagLabel,
		premium,
		statusLabel,
	)

	// Если объявление выложено (активно), показываем дату окончания
	if ad.Status == models.AdStatusActive {
		text += fmt.Sprintf("\n⏱ *Действительно до:* %s", ad.ExpiresAt.Format("02.01.2006 15:04"))
	}

	return text
}

func renderAdPreview(session *adSession) string {
	ad := session.Ad
	if session.DurationDays > 0 {
		ad.ExpiresAt = time.Now().Add(time.Duration(session.DurationDays) * 24 * time.Hour)
	}

	premium := "нет"
	if ad.IsPremium {
		premium = "да"
	}

	categoryLabel := categoryLabels[ad.Category]
	if categoryLabel == "" {
		categoryLabel = ad.Category
	}

	modeLabel := modeLabels[ad.Category][ad.Mode]
	if modeLabel == "" {
		modeLabel = ad.Mode
	}

	tagLabel := tagLabels[ad.Category][ad.Tag]
	if tagLabel == "" {
		tagLabel = ad.Tag
	}

	// Экранируем специальные символы Markdown в тексте объявления
	escapedTitle := escapeMarkdown(ad.Title)
	escapedDesc := escapeMarkdown(ad.Desc)
	escapedUsername := escapeMarkdown(ad.Username)
	escapedClientID := escapeMarkdown(ad.ClientID)

	return fmt.Sprintf(
		"📋 *Предпросмотр объявления*\n\n"+
			"📝 Заголовок: %s\n"+
			"📄 Описание: %s\n"+
			"👤 Контакт: @%s\n"+
			"📂 Категория: %s\n"+
			"🎯 Режим: %s\n"+
			"🏷 Тег: %s\n"+
			"⭐ Премиум: %s\n"+
			"🆔 ID клиента: %s\n"+
			"⏱ Действительно до: %s\n\n"+
			"Подтвердите публикацию:",
		escapedTitle,
		escapedDesc, // Показываем полный текст описания, без обрезки
		escapedUsername,
		categoryLabel,
		modeLabel,
		tagLabel,
		premium,
		escapedClientID,
		ad.ExpiresAt.Format("02.01.2006 15:04"),
	)
}

// escapeMarkdown экранирует специальные символы Markdown для Telegram Bot API
func escapeMarkdown(text string) string {
	// Экранируем специальные символы Markdown: * _ [ ] ( ) ~ ` >
	text = strings.ReplaceAll(text, "*", "\\*")
	text = strings.ReplaceAll(text, "_", "\\_")
	text = strings.ReplaceAll(text, "[", "\\[")
	text = strings.ReplaceAll(text, "]", "\\]")
	text = strings.ReplaceAll(text, "(", "\\(")
	text = strings.ReplaceAll(text, ")", "\\)")
	text = strings.ReplaceAll(text, "~", "\\~")
	text = strings.ReplaceAll(text, "`", "\\`")
	text = strings.ReplaceAll(text, ">", "\\>")
	return text
}

func sendText(bot *tgbotapi.BotAPI, chatID int64, text string) {
	if strings.TrimSpace(text) == "" {
		return
	}
	msg := tgbotapi.NewMessage(chatID, text)
	sentMsg, err := bot.Send(msg)
	if err != nil {
		log.Printf("failed to send message: %v", err)
	} else {
		addBotMessage(chatID, sentMsg.MessageID)
	}
}

func startAdSchedulers(bot *tgbotapi.BotAPI) {
	go func() {
		ticker := time.NewTicker(time.Minute * 30)
		defer ticker.Stop()
		for range ticker.C {
			persistSessionsCleanup()
			processPreExpiry(bot)
			processExpired(bot)
		}
	}()
}

func processPreExpiry(bot *tgbotapi.BotAPI) {
	now := time.Now()
	cutoff := now.Add(24 * time.Hour)

	var ads []models.Ad
	if err := db.DB.Where("status = ? AND expires_at BETWEEN ? AND ? AND pre_expiry_notified = ?", models.AdStatusActive, now, cutoff, false).Find(&ads).Error; err != nil {
		log.Printf("pre-expiry scan failed: %v", err)
		return
	}

	for _, ad := range ads {
		if ad.UserID == 0 {
			continue
		}
		text := fmt.Sprintf("Напоминание: срок действия вашего объявления «%s» истекает %s. Свяжитесь с %s, чтобы продлить размещение.", ad.Title, ad.ExpiresAt.Format("02.01.2006 15:04"), managerHelpLink)
		notifyUser(bot, ad.UserID, text)
		if err := db.DB.Model(&models.Ad{}).Where("id = ?", ad.ID).Update("pre_expiry_notified", true).Error; err != nil {
			log.Printf("pre-expiry flag update failed for ad %d: %v", ad.ID, err)
		}
	}
}

func processExpired(bot *tgbotapi.BotAPI) {
	now := time.Now()

	var ads []models.Ad
	if err := db.DB.Where("status = ? AND expires_at <= ?", models.AdStatusActive, now).Find(&ads).Error; err != nil {
		log.Printf("expiry scan failed: %v", err)
		return
	}

	for _, ad := range ads {
		if err := db.DB.Model(&models.Ad{}).Where("id = ?", ad.ID).Updates(map[string]interface{}{
			"status":              models.AdStatusExpired,
			"pre_expiry_notified": false,
		}).Error; err != nil {
			log.Printf("failed to mark ad %d expired: %v", ad.ID, err)
			continue
		}

		if ad.UserID != 0 {
			text := fmt.Sprintf("Ваше объявление «%s» больше не отображается на бирже. Свяжитесь с %s, чтобы поднять его снова.", ad.Title, managerHelpLink)
			notifyUser(bot, ad.UserID, text)
		}
	}
}
