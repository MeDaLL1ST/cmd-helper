# `cmd-helper`: AI-ассистент в командной строке

![GitHub release (latest by date)](https://img.shields.io/github/v/release/MeDaLL1ST/cmd-helper)
![GitHub Workflow Status](https://img.shields.io/github/actions/workflow/status/MeDaLL1ST/cmd-helper/release.yml)
![Go version](https://img.shields.io/github/go-mod/go-version/MeDaLL1ST/cmd-helper)

Надоело гуглить правильный синтаксис для `find`, `tar` или `ffmpeg`? `cmd-helper` — это утилита, которая исправляет ваши консольные команды с помощью LLM (Large Language Model) прямо в терминале по нажатию одной горячей клавиши.

## Возможности

*   **Мгновенное исправление:** Получите правильный синтаксис команды, не покидая терминал.
*   **Простая интеграция:** Автоматическая настройка для **Zsh** и **Bash**.
*   **Умное исправление:** Утилита сохраняет переменные окружения (`$VAR`) и понимает контекст команды.
*   **Горячая клавиша:** Используйте `Ctrl + A` для вызова ассистента.
*   **Кроссплатформенность:** Работает на macOS (Intel, Apple Silicon) и Linux (amd64, arm64).
*   **Гибкость:** Поддерживает любой LLM-провайдер с OpenAI-совместимым API.

## Установка

Для установки выполните следующую команду в вашем терминале. Скрипт автоматически определит вашу ОС и архитектуру, скачает нужный бинарный файл и настроит вашу оболочку.

```bash
curl -fsSL https://raw.githubusercontent.com/MeDaLL1ST/cmd-helper/main/install.sh | bash
```
**Замечание:** Рекомендуется сначала [ознакомиться с кодом скрипта](https://raw.githubusercontent.com/MeDaLL1ST/cmd-helper/main/install.sh), прежде чем выполнять его.

## Конфигурация

После установки `cmd-helper` требует настройки переменных окружения для доступа к LLM API.

Добавьте следующие переменные в ваш конфигурационный файл (`~/.zshrc`, `~/.bashrc` или `~/.profile`):

```bash
# Пример для ~/.zshrc или ~/.bashrc

# Обязательная переменная: ваш API-токен
export LLM_API_TOKEN="sk-..."

# Опционально: URL для API (по умолчанию используется openai)
export LLM_API_URL="https://api.openai.com/v1/chat/completions"

# Опционально: имя модели (по умолчанию 'o4-mini')
export LLM_MODEL_NAME="имя_вашей_модели"

# Опционально: можно переопределить системный промпт
export LLM_CMD_HELPER_PROMPT_PREFIX="Исправь эту команду: "
```

### Переменные окружения

*   `LLM_API_TOKEN` (**Обязательно**): Ваш секретный ключ для доступа к API.
*   `LLM_API_URL` (*Опционально*): URL эндпоинта. По умолчанию: `https://api.openai.com/v1/chat/completions`.
*   `LLM_MODEL_NAME` (*Опционально*): Имя модели для запросов. По умолчанию: `o4-mini`.
*   `LLM_CMD_HELPER_PROMPT_PREFIX` (*Опционально*): Позволяет полностью переопределить системный промпт, который отправляется модели.

После добавления переменных перезапустите терминал или выполните `source ~/.zshrc` (или `source ~/.bashrc`).

## Использование

1.  Начните вводить в терминале команду, в которой вы не уверены.
2.  Нажмите `Ctrl + A`.
3.  Ваша команда будет мгновенно заменена на исправленную версию.

Вы также можете использовать утилиту напрямую для отладки:
```bash
cmd-helper "gryp 'hello' file.txt"
# Вывод: grep 'hello' file.txt
```

## Сборка из исходников

Если вы хотите собрать проект самостоятельно:

1.  Клонируйте репозиторий:
    ```bash
    git clone https://github.com/MeDaLL1ST/cmd-helper.git
    cd YOUR_REPONAME
    ```

2.  Соберите проект:
    ```bash
    go build
    ```

3.  Переместите бинарный файл в директорию из вашего `PATH`:
    ```bash
    sudo mv cmd-helper /usr/local/bin/
    ```
