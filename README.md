![Go Version](https://img.shields.io/badge/go-1.25-blue)
![License](https://img.shields.io/badge/license-MIT-green)
![Status](https://img.shields.io/badge/status-active-success)

# GitHub Sentinel

AI-powered GitHub repository monitoring and reporting tool.

---

## 🚀 What is GitHub Sentinel?

GitHub Sentinel is an open-source AI agent designed for developers and project managers to efficiently track repository activity.

It periodically fetches updates from subscribed GitHub repositories, aggregates commits, pull requests, and issues, and delivers concise, structured reports through customizable notification channels.

---

## ✨ Features

* 📌 **Subscription Management**

    * Subscribe/unsubscribe to repositories
    * Support for multiple users

* 🔄 **Automated Updates Fetching**

    * Periodic (daily/weekly) sync
    * Tracks commits, pull requests, issues, and releases

* 📊 **Event Aggregation**

    * Merge and deduplicate repository events
    * Categorize updates (feature, fix, docs, etc.)

* 🔔 **Notification System**

    * Email (planned)
    * Slack/Webhook (planned)
    * Extensible notifier interface

* 📝 **Report Generation**

    * Markdown reports
    * AI-powered summaries (coming soon)

* 🧠 **AI Agent (Planned)**

    * Summarize changes
    * Highlight important updates
    * Detect risks and breaking changes

---

## 🏗️ Architecture

```
github-sentinel/
├── cmd/                # Application entrypoint
├── internal/
│   ├── domain/         # Core domain models
│   ├── service/        # Business logic
│   ├── repository/     # Data access layer
│   ├── scheduler/      # Cron jobs
│   ├── github/         # GitHub API client
│   ├── aggregator/     # Event aggregation
│   ├── notifier/       # Notification system
│   └── agent/          # AI agent (future)
├── api/                # HTTP handlers
├── pkg/                # Shared utilities
├── configs/            # Configuration
```

---

## ⚙️ Getting Started

### 1. Clone the repository

```bash
git clone https://github.com/yourname/github-sentinel.git
cd github-sentinel
```

---

### 2. Initialize dependencies

```bash
go mod tidy
```

---

### 3. Run the application

```bash
go run cmd/server/main.go
```

---

## 🧩 Configuration

Edit the config file:

```
configs/config.yaml
```

Example:

```yaml
app:
  name: github-sentinel
  port: 8080
```

---

## 🔌 Future Integrations

* GitHub GraphQL API
* Redis (caching & rate limiting)
* MySQL / PostgreSQL
* Message queue (Kafka / RabbitMQ)
* OpenAI / LLM APIs

---

## 🛠️ Tech Stack

* **Language:** Go
* **Framework:** Gin (planned)
* **Scheduler:** cron
* **Database:** MySQL (planned)
* **Cache:** Redis (planned)

---

## 📈 Roadmap

### Phase 1 (MVP)

* [x] Project scaffolding
* [ ] Subscription system
* [ ] GitHub API integration
* [ ] Basic scheduler

### Phase 2

* [ ] Event persistence (DB)
* [ ] Deduplication strategy
* [ ] Notification system

### Phase 3

* [ ] AI summarization
* [ ] Intelligent alerting
* [ ] Web UI dashboard

---

## 🤝 Contributing

Contributions are welcome!

1. Fork the repo
2. Create your feature branch
3. Commit your changes
4. Open a pull request

---

## 📄 License

MIT License

---

## 💡 Vision

GitHub Sentinel aims to become a **developer productivity assistant**, helping teams stay aligned with fast-moving repositories by turning raw GitHub activity into meaningful insights.

---

## ⭐ Star this project

If you find this project useful, please give it a star!
