package notifier

import (
	"fmt"
	"html"
	"strings"
	"time"
)

// Section texts mirror the website's i18n files (client/src/assets/i18n/*.json).
// When the site copy changes, update it here too.

type yearMonth struct{ y, m int } // m is 1-based

type job struct {
	company, role, location, project string
	start                            yearMonth
	end                              *yearMonth // nil = present
	// Achievements use the site's [[highlight]] markup.
	achievements []string
	tags         []string
}

type project struct {
	title, desc, url string
}

type caseStudy struct {
	title, desc string
	highlights  []string
	path        string // site route, e.g. /projects/rigla
}

type content struct {
	role, tagline string

	aboutTitle, aboutP1, aboutP2                  string
	location, experience, availability, languages string
	degree                                        string
	approachTitle                                 string
	approach                                      [4][2]string // title, desc

	skillsTitle string
	skillGroups [][2]string // label, items

	experienceTitle, experienceComment, present string
	durationY, durationM                        string
	jobs                                        []job

	projectsTitle, featured, more, caseCTA string
	cases                                  []caseStudy
	projects                               []project

	contactTitle, contactText, available string
}

var (
	tagsSurf   = []string{"Go", "PostgreSQL", "Redis", "Kafka", "gRPC", "REST", "Docker", "Prometheus", "Grafana"}
	tagsEvrone = []string{"Go", "PostgreSQL", "Redis", "Kafka", "gRPC", "REST", "SSE", "Docker", "Prometheus", "Grafana", "GitLab CI/CD", "k6"}
	tagsRUDN   = []string{"Go", "PostgreSQL", "REST API", "JWT", "Active Directory"}

	periodSurf   = yearMonth{2025, 5}
	periodEvrone = [2]yearMonth{{2023, 7}, {2025, 4}}
	periodRUDN   = [2]yearMonth{{2021, 11}, {2023, 6}}
)

var contents = map[Lang]content{
	LangEN: {
		role:    "Full-Stack Developer (Go & Angular)",
		tagline: "Go backends · responsive Angular frontends\nHigh-load APIs · PostgreSQL · Kafka",

		aboutTitle: "About",
		aboutP1: "I'm a <b>full-stack developer</b> with <b>5 years of commercial experience</b>. I specialize in <b>high-load microservices</b>: " +
			"API design, caching, PostgreSQL optimization and <b>event-driven architecture on Kafka</b>. On the frontend I build " +
			"<b>fast, responsive web apps in Angular</b> that look right on every screen, from phone to desktop.",
		aboutP2: "I've built systems for products with millions of users — <b>Rostic's (ex-KFC Russia, 1 300+ restaurants)</b> and the " +
			"<b>Rigla pharmacy chain</b>. I'm used to owning a service end to end: <b>from design to production metrics</b>.",
		location:      "Moscow, Russia",
		experience:    "5 years",
		availability:  "Full-time · Freelance",
		languages:     "Arabic (native) · English (C1) · Russian (C1)",
		degree:        "Master's Degree · RUDN University, Moscow · 2023",
		approachTitle: "My approach",
		approach: [4][2]string{
			{"High Load", "5 000 RPS · p99 12 ms"},
			{"Reliability", "Idempotency · Distributed locks"},
			{"Full-Stack", "Go API · Angular UI"},
			{"Metrics-Driven", "Prometheus · Grafana · k6"},
		},

		skillsTitle: "Technical Skills",
		skillGroups: [][2]string{
			{"Languages", "Go · TypeScript · SQL · JavaScript"},
			{"Backend & APIs", "REST API · gRPC · Microservices · Apache Kafka · Event-Driven Architecture · SSE"},
			{"Frontend", "Angular · Responsive Design · HTML5 · CSS3 · SCSS"},
			{"Databases", "PostgreSQL · MongoDB · Redis · Query Optimization & Partitioning"},
			{"Tools & Infrastructure", "Docker · Docker Compose · Git · CI/CD · Prometheus · Grafana · k6 Load Testing · Linux"},
		},

		experienceTitle:   "Experience",
		experienceComment: "5 years of commercial Go development",
		present:           "Present",
		durationY:         "yr",
		durationM:         "mo",
		jobs: []job{
			{
				company: "Surf IT", role: "Golang Developer", location: "Voronezh, Russia",
				project: "Rigla — one of Russia's largest pharmacy chains",
				start:   periodSurf,
				achievements: []string{
					"Designed an API Gateway in Go — a single entry point for the mobile app, website and external systems: JWT authentication, rate limiting, routing. [[~4 000 RPS]] at [[p99 < 50 ms]] during peak periods.",
					"Built the loyalty service (points accrual, discounts, transactions) with idempotency via Redis. Cut discount [[p99 from 120 to 20 ms]] with caching invalidated through Kafka.",
					"Implemented the BOPIS flow (Buy Online, Pick Up In Store): stock reservation with distributed locks in Redis, an order-status state machine with idempotent event processing, and notifications via Kafka. Handles [[~10 000 orders a day]].",
					"Introduced multi-level Redis caching with Kafka-driven invalidation and TTL — [[~90% cache hit ratio]].",
					"Partitioned the loyalty transactions table in PostgreSQL — [[50M+ operations]] without query degradation.",
					"Built a dynamic loyalty-card QR code with anti-forgery protection (HMAC/TOTP signature) and offline generation.",
				},
				tags: tagsSurf,
			},
			{
				company: "Evrone", role: "Golang Developer", location: "Voronezh, Russia",
				project: "Rostic's Platform (ex-KFC Russia) — 1 300+ restaurants, millions of orders",
				start:   periodEvrone[0], end: &periodEvrone[1],
				achievements: []string{
					"Built UserAPI from scratch in Go — the central layer for digital kiosks, kitchen displays and mobile clients (REST and gRPC). [[5 000 RPS]] at [[p99 12 ms]], verified by k6 load testing.",
					"Implemented device registration via pairing codes and token revocation through Redis. New kiosk onboarding went from hours to [[minutes]].",
					"Introduced Redis menu caching with Kafka invalidation. Menu [[p99 dropped from 42 to 3 ms]].",
					"Built an SSE mechanism for menu updates: [[200+ concurrent connections]], automatic reconnection and missed-event recovery.",
					"Set up observability: Prometheus metrics, Grafana dashboards, structured logging with request_id for end-to-end tracing.",
				},
				tags: tagsEvrone,
			},
			{
				company: "RUDN University", role: "Junior Golang Developer", location: "Moscow, Russia",
				project: "Internal university services — 30 000+ students, 2 000+ teachers, up to 10 000+ concurrent users",
				start:   periodRUDN[0], end: &periodRUDN[1],
				achievements: []string{
					"Built an electronic schedule service aggregating data from [[10+ sources]]. Schedule generation went from [[20+ s to ~2 s]].",
					"Built an academic performance service: automatic ranking, JWT authorization, Active Directory integration.",
					"Optimized PostgreSQL queries: indexes and materialized views for heavy reports.",
				},
				tags: tagsRUDN,
			},
		},

		projectsTitle: "Projects",
		featured:      "Featured case studies",
		more:          "More projects",
		caseCTA:       "Read the case study",
		cases: []caseStudy{
			{
				title: "Rigla — Loyalty & BOPIS Backend",
				desc:  "Backend for one of Russia's largest pharmacy chains: API Gateway, loyalty program, dynamic QR cards and Buy Online, Pick Up In Store.",
				highlights: []string{
					"API Gateway: ~4 000 RPS at p99 < 50 ms",
					"Discount p99 cut from 120 to 20 ms",
					"~10 000 BOPIS orders a day · ~90% cache hit ratio",
				},
				path: "/projects/rigla",
			},
			{
				title: "Rostic's Platform (ex-KFC) — UserAPI",
				desc:  "Unified API for digital kiosks, kitchen displays, and mobile clients across 1 300+ restaurants.",
				highlights: []string{
					"5 000 RPS at p99 12 ms, verified with k6",
					"Real-time menu updates via SSE with auto-reconnect",
					"Device pairing and secure token lifecycle via Redis",
				},
				path: "/projects/kfc",
			},
		},
		projects: []project{
			{"Task Management Service", "REST API for team-based task management with role-based access control.", "https://github.com/ramisoul84/task-management"},
			{"Cheatsheet Website", "Personal cheatsheet library for developer skills — searchable and organized by category.", "https://devbrain.ramisuliman.com"},
			{"Encryption Learning Website", "Interactive guide that teaches modern encryption algorithms step by step, with live visualizations.", "https://ramisoul84.github.io/rudn-master/"},
			{"Portfolio Website", "Personal portfolio site for a designer, with dynamic content managed through a headless CMS.", "https://emilsuliman.com"},
		},

		contactTitle: "Have a project in mind? Let's build it.",
		contactText:  "Whether you need a fast Go backend, a responsive Angular frontend, or the whole product end to end, send me a short note about what you're building and I'll reply with next steps.",
		available:    "Available for full-time roles & freelance",
	},

	LangRU: {
		role:    "Full-stack разработчик (Go и Angular)",
		tagline: "Backend на Go · адаптивный frontend на Angular\nВысоконагруженные API · PostgreSQL · Kafka",

		aboutTitle: "Обо мне",
		aboutP1: "Я <b>full-stack разработчик</b>, <b>5 лет коммерческого опыта</b>. Специализируюсь на <b>высоконагруженных микросервисах</b>: " +
			"проектирование API, кэширование, оптимизация PostgreSQL, <b>event-driven архитектура на Kafka</b>. На фронтенде создаю " +
			"<b>быстрые адаптивные веб-приложения на Angular</b>, которые корректно выглядят на любом экране — от телефона до десктопа.",
		aboutP2: "Делал системы для продуктов с миллионами пользователей — <b>Rostic's (ex-KFC Russia, 1 300+ ресторанов)</b> и " +
			"<b>аптечная сеть «Ригла»</b>. Привык отвечать за сервис целиком: <b>от проектирования до метрик в продакшене</b>.",
		location:      "Москва, Россия",
		experience:    "5 лет",
		availability:  "Полная занятость · Фриланс",
		languages:     "Арабский (родной) · Английский (C1) · Русский (C1)",
		degree:        "Магистр · РУДН, Москва · 2023",
		approachTitle: "Мой подход",
		approach: [4][2]string{
			{"Высокая нагрузка", "5 000 RPS · p99 12 мс"},
			{"Надёжность", "Идемпотентность · Распределённые блокировки"},
			{"Full-Stack", "API на Go · UI на Angular"},
			{"Решения на основе метрик", "Prometheus · Grafana · k6"},
		},

		skillsTitle: "Технические навыки",
		skillGroups: [][2]string{
			{"Языки", "Go · TypeScript · SQL · JavaScript"},
			{"Backend и API", "REST API · gRPC · Микросервисы · Apache Kafka · Event-driven архитектура · SSE"},
			{"Frontend", "Angular · Адаптивная вёрстка · HTML5 · CSS3 · SCSS"},
			{"Базы данных", "PostgreSQL · MongoDB · Redis · Оптимизация запросов и партиционирование"},
			{"Инструменты и инфраструктура", "Docker · Docker Compose · Git · CI/CD · Prometheus · Grafana · Нагрузочное тестирование k6 · Linux"},
		},

		experienceTitle:   "Опыт",
		experienceComment: "5 лет коммерческой разработки на Go",
		present:           "настоящее время",
		durationY:         "г.",
		durationM:         "мес.",
		jobs: []job{
			{
				company: "Surf IT", role: "Golang-разработчик", location: "Воронеж, Россия",
				project: "«Ригла» — одна из крупнейших аптечных сетей России",
				start:   periodSurf,
				achievements: []string{
					"Спроектировал API Gateway на Go — единую точку входа для мобильного приложения, сайта и внешних систем: JWT-аутентификация, rate limiting, маршрутизация. [[~4 000 RPS]] при [[p99 < 50 мс]] в пиковые периоды.",
					"Разработал сервис лояльности (начисление баллов, скидки, транзакции) с идемпотентностью через Redis. Снизил [[p99 выдачи скидок со 120 до 20 мс]] за счёт кэширования с инвалидацией через Kafka.",
					"Реализовал флоу BOPIS («Закажи онлайн — забери в магазине»): резервирование остатков с распределёнными блокировками в Redis, конечный автомат статусов заказа с идемпотентной обработкой событий, уведомления через Kafka. Обрабатывает [[~10 000 заказов в день]].",
					"Внедрил многоуровневое кэширование на Redis с инвалидацией через Kafka и TTL — [[Cache Hit Ratio ~90%]].",
					"Партиционировал таблицу транзакций лояльности в PostgreSQL — [[50M+ операций]] без деградации запросов.",
					"Разработал динамический QR-код карты лояльности с защитой от подделки (подпись HMAC/TOTP) и офлайн-генерацией.",
				},
				tags: tagsSurf,
			},
			{
				company: "Evrone", role: "Golang-разработчик", location: "Воронеж, Россия",
				project: "Rostic's Platform (ex-KFC Russia) — 1 300+ ресторанов, миллионы заказов",
				start:   periodEvrone[0], end: &periodEvrone[1],
				achievements: []string{
					"Разработал UserAPI с нуля на Go — центральный слой для цифровых киосков, кухонных дисплеев и мобильных клиентов (REST и gRPC). [[5 000 RPS]] при [[p99 12 мс]], подтверждено нагрузочным тестированием в k6.",
					"Реализовал регистрацию устройств через pairing-коды и отзыв токенов через Redis. Онбординг нового киоска сократился с часов до [[минут]].",
					"Внедрил кэширование меню в Redis с инвалидацией через Kafka. [[p99 выдачи меню снизился с 42 до 3 мс]].",
					"Разработал SSE-механизм обновлений меню: [[200+ одновременных подключений]], автоматическое переподключение и восстановление пропущенных событий.",
					"Настроил observability: метрики в Prometheus, дашборды в Grafana, структурированное логирование с request_id для сквозной трассировки.",
				},
				tags: tagsEvrone,
			},
			{
				company: "РУДН", role: "Junior Golang-разработчик", location: "Москва, Россия",
				project: "Внутренние цифровые сервисы университета — 30 000+ студентов, 2 000+ преподавателей, до 10 000+ одновременных пользователей",
				start:   periodRUDN[0], end: &periodRUDN[1],
				achievements: []string{
					"Разработал сервис электронного расписания с агрегацией данных из [[10+ источников]]. Время формирования расписания сократилось [[с 20+ до ~2 секунд]].",
					"Разработал сервис учёта успеваемости: автоматический расчёт рейтинга, JWT-авторизация, интеграция с Active Directory.",
					"Оптимизировал SQL-запросы в PostgreSQL: индексы, materialized views для тяжёлых отчётов.",
				},
				tags: tagsRUDN,
			},
		},

		projectsTitle: "Проекты",
		featured:      "Кейсы",
		more:          "Другие проекты",
		caseCTA:       "Читать кейс",
		cases: []caseStudy{
			{
				title: "«Ригла» — бэкенд лояльности и BOPIS",
				desc:  "Бэкенд одной из крупнейших аптечных сетей России: API Gateway, программа лояльности, динамические QR-карты и «Закажи онлайн — забери в магазине».",
				highlights: []string{
					"API Gateway: ~4 000 RPS при p99 < 50 мс",
					"p99 выдачи скидок снижен со 120 до 20 мс",
					"~10 000 заказов BOPIS в день · Cache Hit Ratio ~90%",
				},
				path: "/projects/rigla",
			},
			{
				title: "Rostic's Platform (ex-KFC) — UserAPI",
				desc:  "Единый API для цифровых киосков, кухонных дисплеев и мобильных клиентов в 1 300+ ресторанах.",
				highlights: []string{
					"5 000 RPS при p99 12 мс, подтверждено k6",
					"Обновления меню в реальном времени через SSE",
					"Регистрация устройств и жизненный цикл токенов в Redis",
				},
				path: "/projects/kfc",
			},
		},
		projects: []project{
			{"Сервис управления задачами", "REST API для командного управления задачами с ролевым доступом.", "https://github.com/ramisoul84/task-management"},
			{"Сайт-шпаргалка", "Личная библиотека шпаргалок для разработчика — с поиском и разбивкой по категориям.", "https://devbrain.ramisuliman.com"},
			{"Сайт для изучения шифрования", "Интерактивное руководство по современным алгоритмам шифрования — шаг за шагом, с живыми визуализациями.", "https://ramisoul84.github.io/rudn-master/"},
			{"Сайт-портфолио", "Персональное портфолио дизайнера с динамическим контентом из headless CMS.", "https://emilsuliman.com"},
		},

		contactTitle: "Есть идея проекта? Давайте реализуем её.",
		contactText:  "Нужен быстрый backend на Go, адаптивный frontend на Angular или весь продукт целиком — напишите пару слов о задаче, и я отвечу с предложением по следующим шагам.",
		available:    "Открыт к полной занятости и фрилансу",
	},

	LangDE: {
		role:    "Full-Stack-Entwickler (Go & Angular)",
		tagline: "Go-Backends · responsive Angular-Frontends\nHochlast-APIs · PostgreSQL · Kafka",

		aboutTitle: "Über mich",
		aboutP1: "Ich bin <b>Full-Stack-Entwickler</b> mit <b>5 Jahren Berufserfahrung</b>. Mein Schwerpunkt sind <b>hochlastfähige Microservices</b>: " +
			"API-Design, Caching, PostgreSQL-Optimierung und <b>ereignisgesteuerte Architektur mit Kafka</b>. Im Frontend entwickle ich " +
			"<b>schnelle, responsive Web-Apps mit Angular</b>, die auf jedem Bildschirm gut aussehen – vom Smartphone bis zum Desktop.",
		aboutP2: "Ich habe Systeme für Produkte mit Millionen von Nutzern gebaut — <b>Rostic's (ex-KFC Russland, 1 300+ Restaurants)</b> und die " +
			"<b>Apothekenkette Rigla</b>. Ich übernehme Verantwortung für den ganzen Service: <b>vom Design bis zu den Produktionsmetriken</b>.",
		location:      "Moskau, Russland",
		experience:    "5 Jahre",
		availability:  "Festanstellung · Freelance",
		languages:     "Arabisch (Muttersprache) · Englisch (C1) · Russisch (C1)",
		degree:        "Master-Abschluss · RUDN Universität, Moskau · 2023",
		approachTitle: "Mein Ansatz",
		approach: [4][2]string{
			{"Hohe Last", "5 000 RPS · p99 12 ms"},
			{"Zuverlässigkeit", "Idempotenz · Verteilte Locks"},
			{"Full-Stack", "Go-API · Angular-UI"},
			{"Metrikgetrieben", "Prometheus · Grafana · k6"},
		},

		skillsTitle: "Technische Fähigkeiten",
		skillGroups: [][2]string{
			{"Sprachen", "Go · TypeScript · SQL · JavaScript"},
			{"Backend & APIs", "REST API · gRPC · Microservices · Apache Kafka · Ereignisgesteuerte Architektur · SSE"},
			{"Frontend", "Angular · Responsive Design · HTML5 · CSS3 · SCSS"},
			{"Datenbanken", "PostgreSQL · MongoDB · Redis · Query-Optimierung & Partitionierung"},
			{"Tools & Infrastruktur", "Docker · Docker Compose · Git · CI/CD · Prometheus · Grafana · Lasttests mit k6 · Linux"},
		},

		experienceTitle:   "Erfahrung",
		experienceComment: "5 Jahre kommerzielle Go-Entwicklung",
		present:           "heute",
		durationY:         "J.",
		durationM:         "Mon.",
		jobs: []job{
			{
				company: "Surf IT", role: "Golang-Entwickler", location: "Woronesch, Russland",
				project: "Rigla — eine der größten Apothekenketten Russlands",
				start:   periodSurf,
				achievements: []string{
					"API Gateway in Go entworfen — zentraler Einstiegspunkt für Mobile App, Website und externe Systeme: JWT-Authentifizierung, Rate Limiting, Routing. [[~4 000 RPS]] bei [[p99 < 50 ms]] in Spitzenzeiten.",
					"Loyalty-Service (Punktegutschrift, Rabatte, Transaktionen) mit Idempotenz über Redis entwickelt. [[p99 der Rabattausgabe von 120 auf 20 ms]] gesenkt — durch Caching mit Invalidierung über Kafka.",
					"BOPIS-Flow (Online bestellen, im Laden abholen) umgesetzt: Bestandsreservierung mit verteilten Locks in Redis, Zustandsautomat für Bestellstatus mit idempotenter Event-Verarbeitung, Benachrichtigungen über Kafka. Verarbeitet [[~10 000 Bestellungen pro Tag]].",
					"Mehrstufiges Redis-Caching mit Kafka-Invalidierung und TTL eingeführt — [[Cache Hit Ratio ~90 %]].",
					"Loyalty-Transaktionstabelle in PostgreSQL partitioniert — [[50M+ Operationen]] ohne Leistungseinbußen bei Abfragen.",
					"Dynamischen QR-Code für die Kundenkarte mit Fälschungsschutz (HMAC/TOTP-Signatur) und Offline-Generierung entwickelt.",
				},
				tags: tagsSurf,
			},
			{
				company: "Evrone", role: "Golang-Entwickler", location: "Woronesch, Russland",
				project: "Rostic's Platform (ex-KFC Russland) — 1 300+ Restaurants, Millionen Bestellungen",
				start:   periodEvrone[0], end: &periodEvrone[1],
				achievements: []string{
					"UserAPI von Grund auf in Go entwickelt — zentrale Schicht für digitale Kioske, Küchendisplays und mobile Clients (REST und gRPC). [[5 000 RPS]] bei [[p99 12 ms]], per k6-Lasttest bestätigt.",
					"Geräteregistrierung über Pairing-Codes und Token-Widerruf über Redis umgesetzt. Onboarding neuer Kioske von Stunden auf [[Minuten]] verkürzt.",
					"Menü-Caching in Redis mit Kafka-Invalidierung eingeführt. [[p99 der Menüausgabe von 42 auf 3 ms]] gesenkt.",
					"SSE-Mechanismus für Menü-Updates entwickelt: [[200+ gleichzeitige Verbindungen]], automatischer Reconnect und Wiederherstellung verpasster Events.",
					"Observability aufgebaut: Prometheus-Metriken, Grafana-Dashboards, strukturiertes Logging mit request_id für durchgängiges Tracing.",
				},
				tags: tagsEvrone,
			},
			{
				company: "RUDN-Universität", role: "Junior Golang-Entwickler", location: "Moskau, Russland",
				project: "Interne digitale Dienste der Universität — 30 000+ Studierende, 2 000+ Lehrende, bis zu 10 000+ gleichzeitige Nutzer",
				start:   periodRUDN[0], end: &periodRUDN[1],
				achievements: []string{
					"Service für den elektronischen Stundenplan mit Datenaggregation aus [[10+ Quellen]] entwickelt. Erstellungszeit [[von 20+ auf ~2 Sekunden]] reduziert.",
					"Service zur Leistungserfassung entwickelt: automatische Ranking-Berechnung, JWT-Autorisierung, Active-Directory-Integration.",
					"SQL-Abfragen in PostgreSQL optimiert: Indizes und Materialized Views für aufwendige Berichte.",
				},
				tags: tagsRUDN,
			},
		},

		projectsTitle: "Projekte",
		featured:      "Fallstudien",
		more:          "Weitere Projekte",
		caseCTA:       "Fallstudie lesen",
		cases: []caseStudy{
			{
				title: "Rigla — Loyalty- & BOPIS-Backend",
				desc:  "Backend für eine der größten Apothekenketten Russlands: API Gateway, Kundenbindungsprogramm, dynamische QR-Karten und „Online bestellen, im Laden abholen“.",
				highlights: []string{
					"API Gateway: ~4 000 RPS bei p99 < 50 ms",
					"p99 der Rabattausgabe von 120 auf 20 ms gesenkt",
					"~10 000 BOPIS-Bestellungen pro Tag · ~90 % Cache Hit Ratio",
				},
				path: "/projects/rigla",
			},
			{
				title: "Rostic's Platform (ex-KFC) — UserAPI",
				desc:  "Einheitliche API für digitale Kioske, Küchendisplays und mobile Clients in 1 300+ Restaurants.",
				highlights: []string{
					"5 000 RPS bei p99 12 ms, per k6 bestätigt",
					"Echtzeit-Menü-Updates per SSE mit Auto-Reconnect",
					"Geräte-Pairing und sicherer Token-Lebenszyklus über Redis",
				},
				path: "/projects/kfc",
			},
		},
		projects: []project{
			{"Aufgabenverwaltungs-Service", "REST API für teambasierte Aufgabenverwaltung mit rollenbasierter Zugriffskontrolle.", "https://github.com/ramisoul84/task-management"},
			{"Cheatsheet-Website", "Persönliche Cheatsheet-Bibliothek für Entwickler — durchsuchbar und nach Kategorien geordnet.", "https://devbrain.ramisuliman.com"},
			{"Website zum Lernen von Verschlüsselung", "Interaktiver Leitfaden zu modernen Verschlüsselungsalgorithmen — Schritt für Schritt, mit Live-Visualisierungen.", "https://ramisoul84.github.io/rudn-master/"},
			{"Portfolio-Website", "Persönliches Portfolio für einen Designer mit dynamischen Inhalten aus einem Headless CMS.", "https://emilsuliman.com"},
		},

		contactTitle: "Ein Projekt im Kopf? Lass es uns umsetzen.",
		contactText:  "Ob schnelles Go-Backend, responsives Angular-Frontend oder das komplette Produkt – schreib mir kurz, woran du arbeitest, und ich melde mich mit den nächsten Schritten.",
		available:    "Offen für Festanstellung & Freelance",
	},
}

const (
	contactEmail    = "eng.rami.suliman@gmail.com"
	contactTelegram = "https://t.me/ramisoul"
	contactGitHub   = "https://github.com/ramisoul84"
	contactLinkedIn = "https://linkedin.com/in/ramisoul"
)

var months = map[Lang][12]string{
	LangEN: {"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"},
	LangRU: {"Янв.", "Февр.", "Март", "Апр.", "Май", "Июнь", "Июль", "Авг.", "Сент.", "Окт.", "Нояб.", "Дек."},
	LangDE: {"Jan.", "Feb.", "März", "Apr.", "Mai", "Juni", "Juli", "Aug.", "Sept.", "Okt.", "Nov.", "Dez."},
}

func contentFor(l Lang) content {
	if c, ok := contents[l]; ok {
		return c
	}
	return contents[LangEN]
}

// ---------------------------------------------------------------------------
// SECTION RENDERERS
// ---------------------------------------------------------------------------

func renderAbout(l Lang) string {
	c := contentFor(l)
	var b strings.Builder
	fmt.Fprintf(&b, "👨‍💻 <b>%s</b>\n\n", c.aboutTitle)
	b.WriteString(c.aboutP1 + "\n\n" + c.aboutP2 + "\n\n")
	fmt.Fprintf(&b, "📍 %s\n⏳ %s\n💼 %s\n🗣 %s\n🎓 %s\n\n",
		c.location, c.experience, c.availability, c.languages, c.degree)
	fmt.Fprintf(&b, "<b>%s</b>\n", c.approachTitle)
	icons := [4]string{"⚡", "🛡", "🧩", "📈"}
	for i, a := range c.approach {
		fmt.Fprintf(&b, "%s <b>%s</b> — %s\n", icons[i], a[0], a[1])
	}
	return b.String()
}

func renderSkills(l Lang) string {
	c := contentFor(l)
	icons := []string{"🔤", "⚙️", "🎨", "🗄", "🧰"}
	var b strings.Builder
	fmt.Fprintf(&b, "🛠 <b>%s</b>\n", c.skillsTitle)
	for i, g := range c.skillGroups {
		fmt.Fprintf(&b, "\n%s <b>%s</b>\n<code>%s</code>\n", icons[i%len(icons)], html.EscapeString(g[0]), html.EscapeString(g[1]))
	}
	return b.String()
}

func renderJob(l Lang, i int, now time.Time) string {
	c := contentFor(l)
	j := c.jobs[i]

	var b strings.Builder
	fmt.Fprintf(&b, "💼 <b>%s</b> · %d/%d\n<i>%s</i>\n\n", c.experienceTitle, i+1, len(c.jobs), c.experienceComment)
	fmt.Fprintf(&b, "🏢 <b>%s</b> — %s\n", html.EscapeString(j.company), html.EscapeString(j.role))
	fmt.Fprintf(&b, "🗓 %s\n📍 %s\n\n", jobPeriod(l, c, j, now), html.EscapeString(j.location))
	fmt.Fprintf(&b, "<b>%s</b>\n\n", html.EscapeString(j.project))

	b.WriteString("<blockquote expandable>")
	for k, a := range j.achievements {
		if k > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString("▸ " + highlight(a))
	}
	b.WriteString("</blockquote>\n\n")

	b.WriteString("<code>" + html.EscapeString(strings.Join(j.tags, " · ")) + "</code>")
	return b.String()
}

func renderProjects(l Lang, siteURL string) string {
	c := contentFor(l)
	var b strings.Builder
	fmt.Fprintf(&b, "🚀 <b>%s</b>\n\n", c.projectsTitle)
	fmt.Fprintf(&b, "⭐ <b>%s</b>\n\n", c.featured)
	for _, cs := range c.cases {
		fmt.Fprintf(&b, "<b>%s</b>\n%s\n", html.EscapeString(cs.title), html.EscapeString(cs.desc))
		b.WriteString("<blockquote>")
		for k, h := range cs.highlights {
			if k > 0 {
				b.WriteString("\n")
			}
			b.WriteString("✓ " + html.EscapeString(h))
		}
		b.WriteString("</blockquote>\n")
		fmt.Fprintf(&b, "→ <a href=\"%s\">%s</a>\n\n", html.EscapeString(siteURL+cs.path), html.EscapeString(c.caseCTA))
	}
	fmt.Fprintf(&b, "🧪 <b>%s</b>\n", c.more)
	for _, p := range c.projects {
		fmt.Fprintf(&b, "\n• <a href=\"%s\">%s</a> — %s", html.EscapeString(p.url), html.EscapeString(p.title), html.EscapeString(p.desc))
	}
	return b.String()
}

func renderContact(l Lang) string {
	c := contentFor(l)
	return fmt.Sprintf(
		"✉️ <b>%s</b>\n\n%s\n\n🟢 <i>%s</i>\n\n"+
			"📧 <code>%s</code>\n"+
			"💬 <a href=\"%s\">@ramisoul</a>\n"+
			"🐙 <a href=\"%s\">github.com/ramisoul84</a>\n"+
			"💼 <a href=\"%s\">linkedin.com/in/ramisoul</a>",
		html.EscapeString(c.contactTitle), html.EscapeString(c.contactText), html.EscapeString(c.available),
		contactEmail, contactTelegram, contactGitHub, contactLinkedIn,
	)
}

// highlight escapes s and turns the site's [[…]] markers into bold text.
func highlight(s string) string {
	s = html.EscapeString(s)
	s = strings.ReplaceAll(s, "[[", "<b>")
	return strings.ReplaceAll(s, "]]", "</b>")
}

// jobPeriod renders "May 2025 — Present · 1 yr 6 mo", counting months inclusively like the site.
func jobPeriod(l Lang, c content, j job, now time.Time) string {
	m, ok := months[l]
	if !ok {
		m = months[LangEN]
	}
	end := yearMonth{now.Year(), int(now.Month())}
	endLabel := c.present
	if j.end != nil {
		end = *j.end
		endLabel = fmt.Sprintf("%s %d", m[end.m-1], end.y)
	}
	total := (end.y-j.start.y)*12 + (end.m - j.start.m) + 1
	var dur []string
	if y := total / 12; y > 0 {
		dur = append(dur, fmt.Sprintf("%d %s", y, c.durationY))
	}
	if r := total % 12; r > 0 {
		dur = append(dur, fmt.Sprintf("%d %s", r, c.durationM))
	}
	return fmt.Sprintf("%s %d — %s · %s", m[j.start.m-1], j.start.y, endLabel, strings.Join(dur, " "))
}
