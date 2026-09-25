package notifier

import "github.com/go-telegram/bot/models"

func (n *TelegramNotifier) menuKeyboard(isAdmin bool) *models.ReplyKeyboardMarkup {
	rows := [][]models.KeyboardButton{
		{{Text: "About"}, {Text: "Skills"}},
		{{Text: "Experience"}, {Text: "Projects"}},
		{{Text: "Contact"}},
	}
	if isAdmin {
		rows = append(rows, []models.KeyboardButton{{Text: "/stats"}, {Text: "/visits"}})
	}
	return &models.ReplyKeyboardMarkup{
		Keyboard:              rows,
		ResizeKeyboard:        true,
		InputFieldPlaceholder: "Choose an option…",
	}
}
