package notifier

import (
	"fmt"
	"html"
	"strings"
)

func resolveSection(text string) (string, bool) {
	key := strings.ToLower(strings.TrimSpace(text))
	switch key {
	case "about", "skills", "experience", "projects", "contact":
		return key, true
	}
	return "", false
}

func sectionContent(key string) string {
	switch key {
	case "about":
		return "👨‍💻 <b>About</b>\n\n" +
			"Full-stack developer with 5+ years of experience. I build high-load backends in Go and responsive frontends in Angular.\n\n" +
			"🌐 <a href=\"https://ramisuliman.com\">ramisuliman.com</a>"
	case "skills":
		return "🛠 <b>Skills</b>\n\n" +
			"<b>Languages</b>\nGo · TypeScript · JavaScript\n\n" +
			"<b>Backend</b>\nREST · gRPC · Microservices · Kafka\n\n" +
			"<b>Databases</b>\nPostgreSQL · MongoDB · Redis\n\n" +
			"<b>Tools</b>\nDocker · Kubernetes · Prometheus · Grafana"
	case "experience":
		return "💼 <b>Experience</b>\n\n" +
			"<b>Friflex · Senior Golang Developer</b>\nMar 2025 — Present · Moscow\n" +
			"Bristol Retail — loyalty backend\n• 12 000 RPS at p99 &lt; 25 ms\n\n" +
			"<b>Evrone · Golang Developer</b>\nJul 2023 — Feb 2025\n" +
			"Rostic's Platform (ex-KFC) — UserAPI\n• 5 000 RPS at p99 12 ms\n\n" +
			"<b>RUDN University</b>\nNov 2021 — Jun 2023\n• 30 000+ students served"
	case "projects":
		return "📦 <b>Projects</b>\n\n" +
			"<b>Bristol Retail</b> — backend for 7 000+ stores, 10M+ app users.\n" +
			"→ <a href=\"https://ramisuliman.com/projects/bristol\">Case study</a>\n\n" +
			"<b>Rostic's Platform (ex-KFC)</b> — UserAPI for 1 300+ restaurants.\n" +
			"→ <a href=\"https://ramisuliman.com/projects/kfc\">Case study</a>"
	case "contact":
		email := "eng.rami.suliman@gmail.com"
		return fmt.Sprintf(
			"✉️ <b>Contact</b>\n\n"+
				"📧 <a href=\"mailto:%s\">%s</a>\n"+
				"💬 Telegram: @ramisoul\n"+
				"🐙 <a href=\"https://github.com/ramisoul84\">github.com/ramisoul84</a>",
			html.EscapeString(email), html.EscapeString(email),
		)
	}
	return ""
}
