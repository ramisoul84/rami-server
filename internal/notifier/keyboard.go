package notifier

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/go-telegram/bot/models"
)

// Callback data layout (Telegram caps it at 64 bytes):
//
//	v:<view>[:<arg>]   public screen, edited in place (v:exp:1 = second job)
//	cv                 send the resume PDF
//	lang:<code>        switch language
//	a:<view>[:<arg>]   admin screen (see admin.go)
//	…:n suffix         render as a new message instead of editing (used under alerts)
//	noop               inert button (page counter)
const (
	viewMenu       = "menu"
	viewAbout      = "about"
	viewSkills     = "skills"
	viewExperience = "exp"
	viewProjects   = "proj"
	viewContact    = "contact"
	viewLanguage   = "lang"

	cbResume = "cv"
	cbNoop   = "noop"
)

// screen is one rendered bot "page": its HTML text and inline keyboard.
type screen struct {
	text string
	kb   *models.InlineKeyboardMarkup
}

func btn(text, data string) models.InlineKeyboardButton {
	return models.InlineKeyboardButton{Text: text, CallbackData: data}
}

func urlBtn(text, url string) models.InlineKeyboardButton {
	return models.InlineKeyboardButton{Text: text, URL: url}
}

func keyboard(rows ...[]models.InlineKeyboardButton) *models.InlineKeyboardMarkup {
	return &models.InlineKeyboardMarkup{InlineKeyboard: rows}
}

func row(buttons ...models.InlineKeyboardButton) []models.InlineKeyboardButton { return buttons }

func viewData(view string, args ...string) string {
	return strings.Join(append([]string{"v", view}, args...), ":")
}

// ---------------------------------------------------------------------------
// PUBLIC SCREENS
// ---------------------------------------------------------------------------

func (n *TelegramNotifier) menuKeyboard(l Lang, isAdmin bool) *models.InlineKeyboardMarkup {
	u := uiFor(l)
	website := urlBtn(u.btnWebsite, n.siteURL)
	website.Style = "primary"
	rows := [][]models.InlineKeyboardButton{
		row(btn(u.btnAbout, viewData(viewAbout)), btn(u.btnSkills, viewData(viewSkills))),
		row(btn(u.btnExperience, viewData(viewExperience, "0")), btn(u.btnProjects, viewData(viewProjects))),
		row(btn(u.btnContact, viewData(viewContact)), btn(u.btnResume, cbResume)),
		row(website),
		row(btn(u.btnLanguage, viewData(viewLanguage))),
	}
	if isAdmin {
		rows = append(rows, row(btn(u.btnAdmin, "a:home")))
	}
	return keyboard(rows...)
}

func (n *TelegramNotifier) menuRow(l Lang) []models.InlineKeyboardButton {
	return row(btn(uiFor(l).btnMenu, viewData(viewMenu)))
}

// publicScreen renders a public view. ok is false for an unknown view.
func (n *TelegramNotifier) publicScreen(l Lang, view, arg string, isAdmin bool) (screen, bool) {
	u := uiFor(l)
	c := contentFor(l)

	switch view {
	case viewMenu:
		return screen{u.menuHint, n.menuKeyboard(l, isAdmin)}, true

	case viewAbout:
		return screen{renderAbout(l), keyboard(
			row(btn(u.btnExperience, viewData(viewExperience, "0")), btn(u.btnResume, cbResume)),
			n.menuRow(l),
		)}, true

	case viewSkills:
		return screen{renderSkills(l), keyboard(
			row(btn(u.btnProjects, viewData(viewProjects)), btn(u.btnExperience, viewData(viewExperience, "0"))),
			n.menuRow(l),
		)}, true

	case viewExperience:
		total := len(c.jobs)
		i, err := strconv.Atoi(arg)
		if err != nil || i < 0 || i >= total {
			i = 0
		}
		prev, next := (i-1+total)%total, (i+1)%total
		return screen{renderJob(l, i, n.now().In(n.loc)), keyboard(
			row(
				btn("◀", viewData(viewExperience, strconv.Itoa(prev))),
				btn(fmt.Sprintf("%d / %d", i+1, total), cbNoop),
				btn("▶", viewData(viewExperience, strconv.Itoa(next))),
			),
			row(btn(u.btnProjects, viewData(viewProjects)), btn(u.btnResume, cbResume)),
			n.menuRow(l),
		)}, true

	case viewProjects:
		return screen{renderProjects(l, n.siteURL), keyboard(
			row(urlBtn(u.btnCaseRigla, n.siteURL+"/projects/rigla"), urlBtn(u.btnCaseKFC, n.siteURL+"/projects/kfc")),
			row(urlBtn("🐙 GitHub", contactGitHub)),
			n.menuRow(l),
		)}, true

	case viewContact:
		message := urlBtn(u.btnTelegram, contactTelegram)
		message.Style = "primary"
		return screen{renderContact(l), keyboard(
			row(message),
			row(models.InlineKeyboardButton{Text: u.btnEmail, CopyText: &models.CopyTextButton{Text: contactEmail}}),
			row(urlBtn("🐙 GitHub", contactGitHub), urlBtn("💼 LinkedIn", contactLinkedIn)),
			row(btn(u.btnResume, cbResume)),
			n.menuRow(l),
		)}, true

	case viewLanguage:
		var buttons []models.InlineKeyboardButton
		for _, code := range langs {
			label := uiFor(code).langName
			if code == l {
				label = "✓ " + label
			}
			buttons = append(buttons, btn(label, "lang:"+string(code)))
		}
		return screen{u.languagePrompt, keyboard(buttons, n.menuRow(l))}, true
	}
	return screen{}, false
}

func (n *TelegramNotifier) welcomeScreen(l Lang, firstName string, isAdmin bool) screen {
	u := uiFor(l)
	c := contentFor(l)
	return screen{
		text: fmt.Sprintf(u.welcome, escape(firstName), escape(c.role), escape(c.tagline)),
		kb:   n.menuKeyboard(l, isAdmin),
	}
}
