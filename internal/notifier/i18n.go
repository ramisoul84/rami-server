package notifier

import "strings"

// Lang is a language the public side of the bot speaks — the same set as the website.
type Lang string

const (
	LangEN Lang = "en"
	LangRU Lang = "ru"
	LangDE Lang = "de"
)

var langs = []Lang{LangEN, LangRU, LangDE}

// langFromCode maps a Telegram language_code ("ru", "de-AT", "uk"…) to a supported language.
func langFromCode(code string) Lang {
	code = strings.ToLower(code)
	switch {
	case strings.HasPrefix(code, "ru"), strings.HasPrefix(code, "uk"),
		strings.HasPrefix(code, "be"), strings.HasPrefix(code, "kk"):
		return LangRU
	case strings.HasPrefix(code, "de"):
		return LangDE
	default:
		return LangEN
	}
}

func parseLang(s string) (Lang, bool) {
	for _, l := range langs {
		if string(l) == s {
			return l, true
		}
	}
	return "", false
}

// ui holds the bot's own strings (section texts live in content.go).
type ui struct {
	langName string // shown on the language picker, e.g. "🇬🇧 English"

	welcome  string // %s = first name
	menuHint string

	btnAbout, btnSkills, btnExperience, btnProjects, btnContact string
	btnResume, btnWebsite, btnLanguage, btnMenu, btnAdmin       string
	btnEmail, btnTelegram, btnCaseRigla, btnCaseKFC             string

	resumeCaption, resumeFailed string
	languagePrompt, languageSet string
	unknown                     string

	cmdStart, cmdMenu, cmdResume, cmdContact, cmdLanguage string

	description, shortDescription string
}

var uis = map[Lang]ui{
	LangEN: {
		langName: "🇬🇧 English",
		welcome: "👋 Hi <b>%s</b>, I'm Rami's portfolio bot.\n\n" +
			"<b>Rami Suliman</b> — %s\n<i>%s</i>\n\n" +
			"Explore his work right here, grab the resume, or get in touch 👇",
		menuHint: "🏠 <b>Menu</b>\n\nWhat would you like to see?",

		btnAbout: "👨‍💻 About", btnSkills: "🛠 Skills", btnExperience: "💼 Experience", btnProjects: "🚀 Projects",
		btnContact: "✉️ Contact", btnResume: "📄 Resume", btnWebsite: "🌐 Open website", btnLanguage: "🌍 Language",
		btnMenu: "🏠 Menu", btnAdmin: "👑 Admin panel",
		btnEmail: "📧 Copy email", btnTelegram: "💬 Message Rami", btnCaseRigla: "📖 Rigla case", btnCaseKFC: "📖 Rostic's case",

		resumeCaption:  "📄 <b>Rami Suliman</b> — resume\n%s",
		resumeFailed:   "Couldn't attach the file right now — here's the direct link:\n%s",
		languagePrompt: "🌍 <b>Choose a language</b>",
		languageSet:    "Language: English",
		unknown:        "I'm a menu-driven bot 🙂 Pick a section below.",

		cmdStart: "Welcome & menu", cmdMenu: "Open the menu", cmdResume: "Get the resume (PDF)",
		cmdContact: "Contact details", cmdLanguage: "Change language",

		description: "Rami Suliman — Full-Stack Developer (Go & Angular).\n\n" +
			"Browse experience, skills and case studies, download the resume, or get in touch — all from Telegram.",
		shortDescription: "Rami Suliman's portfolio: Go & Angular developer. Experience, projects, resume.",
	},
	LangRU: {
		langName: "🇷🇺 Русский",
		welcome: "👋 Привет, <b>%s</b>! Я бот-портфолио Рами.\n\n" +
			"<b>Рами Сулиман</b> — %s\n<i>%s</i>\n\n" +
			"Смотрите проекты прямо здесь, скачайте резюме или напишите 👇",
		menuHint: "🏠 <b>Меню</b>\n\nЧто хотите посмотреть?",

		btnAbout: "👨‍💻 Обо мне", btnSkills: "🛠 Навыки", btnExperience: "💼 Опыт", btnProjects: "🚀 Проекты",
		btnContact: "✉️ Контакты", btnResume: "📄 Резюме", btnWebsite: "🌐 Открыть сайт", btnLanguage: "🌍 Язык",
		btnMenu: "🏠 Меню", btnAdmin: "👑 Админ-панель",
		btnEmail: "📧 Скопировать email", btnTelegram: "💬 Написать Рами", btnCaseRigla: "📖 Кейс «Ригла»", btnCaseKFC: "📖 Кейс Rostic's",

		resumeCaption:  "📄 <b>Рами Сулиман</b> — резюме\n%s",
		resumeFailed:   "Не получилось прикрепить файл — вот прямая ссылка:\n%s",
		languagePrompt: "🌍 <b>Выберите язык</b>",
		languageSet:    "Язык: русский",
		unknown:        "Я работаю через меню 🙂 Выберите раздел ниже.",

		cmdStart: "Приветствие и меню", cmdMenu: "Открыть меню", cmdResume: "Скачать резюме (PDF)",
		cmdContact: "Контакты", cmdLanguage: "Сменить язык",

		description: "Рами Сулиман — Full-stack разработчик (Go и Angular).\n\n" +
			"Опыт, навыки и кейсы, резюме и контакты — прямо в Telegram.",
		shortDescription: "Портфолио Рами Сулимана: разработчик на Go и Angular. Опыт, проекты, резюме.",
	},
	LangDE: {
		langName: "🇩🇪 Deutsch",
		welcome: "👋 Hallo <b>%s</b>, ich bin Ramis Portfolio-Bot.\n\n" +
			"<b>Rami Suliman</b> — %s\n<i>%s</i>\n\n" +
			"Projekte direkt hier ansehen, Lebenslauf holen oder Kontakt aufnehmen 👇",
		menuHint: "🏠 <b>Menü</b>\n\nWas möchtest du sehen?",

		btnAbout: "👨‍💻 Über mich", btnSkills: "🛠 Fähigkeiten", btnExperience: "💼 Erfahrung", btnProjects: "🚀 Projekte",
		btnContact: "✉️ Kontakt", btnResume: "📄 Lebenslauf", btnWebsite: "🌐 Website öffnen", btnLanguage: "🌍 Sprache",
		btnMenu: "🏠 Menü", btnAdmin: "👑 Admin-Panel",
		btnEmail: "📧 E-Mail kopieren", btnTelegram: "💬 Rami schreiben", btnCaseRigla: "📖 Fallstudie Rigla", btnCaseKFC: "📖 Fallstudie Rostic's",

		resumeCaption:  "📄 <b>Rami Suliman</b> — Lebenslauf\n%s",
		resumeFailed:   "Die Datei konnte gerade nicht angehängt werden — hier ist der direkte Link:\n%s",
		languagePrompt: "🌍 <b>Sprache wählen</b>",
		languageSet:    "Sprache: Deutsch",
		unknown:        "Ich funktioniere über das Menü 🙂 Wähle unten einen Bereich.",

		cmdStart: "Begrüßung & Menü", cmdMenu: "Menü öffnen", cmdResume: "Lebenslauf (PDF)",
		cmdContact: "Kontaktdaten", cmdLanguage: "Sprache ändern",

		description: "Rami Suliman — Full-Stack-Entwickler (Go & Angular).\n\n" +
			"Erfahrung, Fähigkeiten und Fallstudien, Lebenslauf und Kontakt — direkt in Telegram.",
		shortDescription: "Portfolio von Rami Suliman: Go- & Angular-Entwickler. Erfahrung, Projekte, Lebenslauf.",
	},
}

func uiFor(l Lang) ui {
	if u, ok := uis[l]; ok {
		return u
	}
	return uis[LangEN]
}
