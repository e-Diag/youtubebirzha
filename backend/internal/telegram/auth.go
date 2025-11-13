package telegram

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

type TelegramUser struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name,omitempty"`
	Username  string `json:"username,omitempty"`
	Language  string `json:"language_code,omitempty"`
}

// ValidateInitData проверяет подпись Telegram Mini App init_data
// Возвращает map с данными пользователя и true, если подпись валидна
func ValidateInitData(initData, botToken string) (map[string]string, bool) {
	if initData == "" || botToken == "" {
		return nil, false
	}

	// Парсим init_data
	params, err := url.ParseQuery(initData)
	if err != nil {
		return nil, false
	}

	// Извлекаем hash
	hash := params.Get("hash")
	if hash == "" {
		return nil, false
	}

	// Удаляем hash из параметров для проверки подписи
	params.Del("hash")

	// Сортируем параметры по ключу
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Формируем data-check-string
	var dataCheckString strings.Builder
	for i, k := range keys {
		if i > 0 {
			dataCheckString.WriteString("\n")
		}
		dataCheckString.WriteString(k)
		dataCheckString.WriteString("=")
		dataCheckString.WriteString(params.Get(k))
	}

	// Вычисляем секретный ключ
	secretKey := hmac.New(sha256.New, []byte("WebAppData"))
	secretKey.Write([]byte(botToken))

	// Вычисляем HMAC
	h := hmac.New(sha256.New, secretKey.Sum(nil))
	h.Write([]byte(dataCheckString.String()))
	calculatedHash := hex.EncodeToString(h.Sum(nil))

	// Сравниваем хеши
	if calculatedHash != hash {
		return nil, false
	}

	// Проверяем время (auth_date не должен быть старше 24 часов)
	authDateStr := params.Get("auth_date")
	if authDateStr != "" {
		authDate, err := strconv.ParseInt(authDateStr, 10, 64)
		if err == nil {
			authTime := time.Unix(authDate, 0)
			if time.Since(authTime) > 24*time.Hour {
				return nil, false
			}
		}
	}

	// Извлекаем данные пользователя
	result := make(map[string]string)
	userStr := params.Get("user")
	if userStr != "" {
		result["user"] = userStr
		
		// Парсим JSON user объекта
		var user TelegramUser
		if err := json.Unmarshal([]byte(userStr), &user); err == nil {
			result["user_id"] = strconv.FormatInt(user.ID, 10)
			if user.Username != "" {
				result["username"] = user.Username
			}
		}
	}

	// Сохраняем все параметры
	for k, v := range params {
		if len(v) > 0 {
			result[k] = v[0]
		}
	}

	return result, true
}

// ExtractUserID извлекает user_id из валидированных данных
func ExtractUserID(data map[string]string) (int64, error) {
	userIDStr := data["user_id"]
	if userIDStr == "" {
		return 0, fmt.Errorf("user_id not found")
	}
	
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid user_id: %w", err)
	}
	
	return userID, nil
}

// ExtractUsername извлекает username из валидированных данных
func ExtractUsername(data map[string]string) string {
	return data["username"]
}

