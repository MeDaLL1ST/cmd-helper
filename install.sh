#!/bin/bash
set -e # Выходить из скрипта при любой ошибке

# --- НАСТРОЙКИ ---
GITHUB_USER="MeDaLL1ST"
GITHUB_REPO="cmd-helper"
CMD_NAME="cmd-helper"
INSTALL_DIR="/usr/local/bin"
# --- КОНЕЦ НАСТРОЕК ---

# Функция для определения системы
get_arch() {
    local os_type
    local arch_type
    os_type=$(uname -s | tr '[:upper:]' '[:lower:]')
    arch_type=$(uname -m)

    case "$arch_type" in
        x86_64|amd64)
            arch_type="amd64"
            ;;
        aarch64|arm64)
            arch_type="arm64"
            ;;
        *)
            echo "Ошибка: Неподдерживаемая архитектура: $arch_type"
            exit 1
            ;;
    esac
    echo "${os_type}-${arch_type}"
}

# 1. Определяем архитектуру и ОС
PLATFORM=$(get_arch)
BINARY_NAME="${CMD_NAME}-${PLATFORM}"
echo "Определена платформа: $PLATFORM"

# 2. Находим URL последнего релиза
echo "Поиск последнего релиза в ${GITHUB_USER}/${GITHUB_REPO}..."
LATEST_RELEASE_URL="https://api.github.com/repos/${GITHUB_USER}/${GITHUB_REPO}/releases/latest"

DOWNLOAD_URL=$(curl -sSL $LATEST_RELEASE_URL | grep "browser_download_url.*${BINARY_NAME}" | cut -d '"' -f 4)

if [ -z "$DOWNLOAD_URL" ]; then
    echo "Ошибка: Не удалось найти бинарный файл для вашей платформы ($PLATFORM) в последнем релизе."
    exit 1
fi

# 3. Скачиваем и устанавливаем бинарный файл
echo "Скачивание $DOWNLOAD_URL..."
TEMP_FILE=$(mktemp)
curl -L -o "$TEMP_FILE" "$DOWNLOAD_URL"
chmod +x "$TEMP_FILE"

echo "Установка ${CMD_NAME} в ${INSTALL_DIR}..."
if [ -w "$INSTALL_DIR" ]; then
    mv "$TEMP_FILE" "${INSTALL_DIR}/${CMD_NAME}"
else
    echo "Требуются права суперпользователя для установки в ${INSTALL_DIR}."
    sudo mv "$TEMP_FILE" "${INSTALL_DIR}/${CMD_NAME}"
fi

# 4. Настройка оболочки (zsh или bash)
SHELL_TYPE=$(basename "$SHELL")
CONFIG_FILE=""
CONFIG_BLOCK=""

if [ "$SHELL_TYPE" = "zsh" ]; then
    CONFIG_FILE="$HOME/.zshrc"
    CONFIG_BLOCK=$(cat <<'EOF'

# --- cmd-helper config ---
fix_current_command_widget() {
    local original_buffer=$BUFFER
    local corrected_command
    corrected_command=$(cmd-helper "$BUFFER" 2> /tmp/cmd_helper_zle_error.log)
    if [[ $? -eq 0 ]] && [[ -n "$corrected_command" ]]; then
        BUFFER=$corrected_command
        CURSOR=${#BUFFER}
    else
        BUFFER=$original_buffer
        zle send-break
    fi
    zle redisplay
}
zle -N fix_current_command_widget
bindkey '^a' fix_current_command_widget # Используем Control-a
# --- end cmd-helper config ---
EOF
)
elif [ "$SHELL_TYPE" = "bash" ]; then
    CONFIG_FILE="$HOME/.bashrc"
    # Для bash может понадобиться добавить это в .bash_profile или .profile, если .bashrc не загружается для логин-шеллов
    if [ ! -f "$HOME/.bashrc" ] && [ -f "$HOME/.bash_profile" ]; then
        CONFIG_FILE="$HOME/.bash_profile"
    elif [ ! -f "$HOME/.bashrc" ] && [ -f "$HOME/.profile" ]; then
        CONFIG_FILE="$HOME/.profile"
    fi
    CONFIG_BLOCK=$(cat <<'EOF'

# --- cmd-helper config ---
_fix_current_bash_command_ctrl_a() {
    local original_line=$READLINE_LINE
    local original_point=$READLINE_POINT
    local corrected_command
    corrected_command=$(cmd-helper "$READLINE_LINE" 2> /tmp/cmd_helper_bash_error.log)
    if [[ $? -eq 0 ]] && [[ -n "$corrected_command" ]]; then
        READLINE_LINE="$corrected_command"
        READLINE_POINT=${#corrected_command}
    else
        READLINE_LINE="$original_line"
        READLINE_POINT="$original_point"
    fi
}
bind -x '"\C-a": _fix_current_bash_command_ctrl_a'
# --- end cmd-helper config ---
EOF
)
else
    echo "Предупреждение: Не удалось определить оболочку или ваша оболочка ($SHELL_TYPE) не поддерживается для авто-настройки."
    echo "Установка завершена. Настройте горячую клавишу вручную."
    exit 0
fi

# Проверяем, не добавлен ли уже блок
if ! grep -q "# --- cmd-helper config ---" "$CONFIG_FILE"; then
    echo "Добавление конфигурации в $CONFIG_FILE..."
    echo "$CONFIG_BLOCK" >> "$CONFIG_FILE"
    echo "Конфигурация добавлена."
else
    echo "Конфигурация уже существует в $CONFIG_FILE. Пропускаем."
fi

echo ""
echo "Установка успешно завершена!"
echo "Пожалуйста, перезапустите ваш терминал или выполните 'source ${CONFIG_FILE}' для применения изменений."
echo "После этого нажмите Ctrl+A для исправления текущей введенной команды."
