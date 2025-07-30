package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

// Message структура для сообщений в API
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func SendMessageToAPI(apiURL, apiBearerToken string, model string, messages []Message) (string, error) {
	reqBody := struct {
		Model    string    `json:"model"`
		Messages []Message `json:"messages"`
		Stream   bool      `json:"stream"`
	}{
		Model:    model,
		Messages: messages,
		Stream:   true,
	}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("ошибка кодирования запроса: %w", err)
	}
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("ошибка создания запроса: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiBearerToken))
	req.Header.Set("Accept", "application/json, text/event-stream") // Принимаем и то, и другое

	client := &http.Client{Timeout: 180 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("ошибка отправки запроса: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return "", fmt.Errorf("API вернул статус %d, и не удалось прочитать тело ошибки: %w", resp.StatusCode, readErr)
		}
		var errorResp struct {
			Error struct {
				Detail  string `json:"detail"`
				Message string `json:"message"`
				Type    string `json:"type"`
				Code    string `json:"code"`
			} `json:"error"`
		}
		errMsg := string(body)
		if json.Unmarshal(body, &errorResp) == nil && (errorResp.Error.Detail != "" || errorResp.Error.Message != "") {
			if errorResp.Error.Detail != "" {
				errMsg = errorResp.Error.Detail
			} else {
				errMsg = fmt.Sprintf("Type: %s, Message: %s, Code: %s", errorResp.Error.Type, errorResp.Error.Message, errorResp.Error.Code)
			}
			if strings.Contains(errorResp.Error.Detail, "image input is not supported") || strings.Contains(errorResp.Error.Message, "image input is not supported") {
				return "", fmt.Errorf("модель не поддерживает обработку изображений (детали: %s)", errMsg)
			}
		}
		return "", fmt.Errorf("API вернул статус %d: %s", resp.StatusCode, errMsg)
	}

	contentType := resp.Header.Get("Content-Type")
	if strings.Contains(contentType, "text/event-stream") {
		var fullContent strings.Builder
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "data:") {
				jsonData := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
				if jsonData == "[DONE]" {
					break
				}
				if jsonData == "" {
					continue
				}
				var chunkResp struct {
					Choices []struct {
						Delta struct {
							Content string `json:"content"`
						} `json:"delta"`
					} `json:"choices"`
				}
				err := json.Unmarshal([]byte(jsonData), &chunkResp)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Предупреждение: не удалось разобрать чанк '%s': %v\n", jsonData, err)
					continue
				}
				if len(chunkResp.Choices) > 0 && chunkResp.Choices[0].Delta.Content != "" {
					fullContent.WriteString(chunkResp.Choices[0].Delta.Content)
				}
			}
		}
		if err := scanner.Err(); err != nil {
			accumulated := fullContent.String()
			if accumulated != "" {
				return accumulated, fmt.Errorf("ошибка чтения потока ПОСЛЕ получения части данных: %w", err)
			}
			return "", fmt.Errorf("ошибка чтения потока: %w", err)
		}
		return fullContent.String(), nil
	} else if strings.Contains(contentType, "application/json") {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", fmt.Errorf("ошибка чтения тела не-потокового ответа: %w", err)
		}
		var apiResp struct {
			Choices []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			} `json:"choices"`
			Error *struct {
				Detail  string `json:"detail"`
				Message string `json:"message"`
				Type    string `json:"type"`
				Code    string `json:"code"`
			} `json:"error,omitempty"`
		}
		err = json.Unmarshal(body, &apiResp)
		if err != nil {
			return "", fmt.Errorf("ошибка разбора JSON ответа: %w (Тело: %s)", err, string(body))
		}
		if apiResp.Error != nil {
			errMsg := fmt.Sprintf("Type: %s, Message: %s, Code: %s, Detail: %s",
				apiResp.Error.Type, apiResp.Error.Message, apiResp.Error.Code, apiResp.Error.Detail)
			if strings.Contains(apiResp.Error.Message, "image input is not supported") || strings.Contains(apiResp.Error.Detail, "image input is not supported") {
				return "", fmt.Errorf("модель не поддерживает обработку изображений (API Error: %s)", errMsg)
			}
			return "", fmt.Errorf("API вернул ошибку в JSON: %s", errMsg)
		}
		if len(apiResp.Choices) == 0 || apiResp.Choices[0].Message.Content == "" {
			return "", fmt.Errorf("API вернул JSON ответ без контента в первом choice")
		}
		return apiResp.Choices[0].Message.Content, nil
	} else {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("неожиданный Content-Type ответа: %s. Тело: %s", contentType, string(bodyBytes))
	}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s \"<command_to_fix>\"\n", os.Args[0])
		os.Exit(1)
	}
	commandToFix := os.Args[1]

	// Получение конфигурации из переменных окружения
	apiToken := os.Getenv("LLM_API_TOKEN")
	if apiToken == "" {
		fmt.Fprintf(os.Stderr, "Ошибка: Переменная окружения LLM_API_TOKEN не установлена.\n")
		os.Exit(1)
	}

	modelName := os.Getenv("LLM_MODEL_NAME")
	if modelName == "" {
		modelName = "o4-mini" // Модель по умолчанию
	}
	fmt.Fprintf(os.Stderr, " | Query to: %s\n", modelName)

	apiURL := os.Getenv("LLM_API_URL")
	if apiURL == "" {
		apiURL = "https://api.openai.com/v1/chat/completions" // URL по умолчанию
		fmt.Fprintf(os.Stderr, "Предупреждение: LLM_API_URL не установлена, используется по умолчанию: %s\n", apiURL)
	}

	defaultPromptPrefix := "Пожалуйста, исправь следующую команду командной строки, чтобы она была синтаксически правильной и выполняла предполагаемое действие. Если команда уже выглядит корректной, верни ее без изменений. Если команда содержит переменные окружения (например, $VAR или %VAR%), сохрани их. Верни только исправленную команду, без каких-либо объяснений или дополнительного текста.\n\nНеправильная команда: "

	// Можно переопределять промпт через переменную окружения
	promptPrefix := os.Getenv("LLM_CMD_HELPER_PROMPT_PREFIX")
	var flag bool = false
	if promptPrefix == "" {
		flag = true
		promptPrefix = defaultPromptPrefix
	}

	messages := []Message{
		{
			Role:    "system",
			Content: "Ты — полезный ассистент, который помогает исправлять ошибки в командах командной строки.",
		},
		{
			Role:    "user",
			Content: promptPrefix + commandToFix,
		},
	}

	correctedCommand, err := SendMessageToAPI(apiURL, apiToken, modelName, messages)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка при запросе к LLM: %v\n", err)
		// В случае ошибки, чтобы не стирать введенную команду, выведем ее обратно
		fmt.Print(commandToFix)
		os.Exit(1)
	}
	if flag {
		reThink := regexp.MustCompile(`(?is)<think[^>]*>.*?</think>`)
		cleaned := reThink.ReplaceAllString(correctedCommand, "")
		trimmedCommand := strings.TrimSpace(cleaned)

		trimmedCommand = strings.ReplaceAll(trimmedCommand, "\r\n", " ") // Для Windows
		trimmedCommand = strings.ReplaceAll(trimmedCommand, "\n", " ")   // Для Unix/Linux/macOS
		trimmedCommand = strings.ReplaceAll(trimmedCommand, "\r", " ")   // Для стар, "\r\n", " ") // Для Windows
		trimmedCommand = strings.ReplaceAll(trimmedCommand, "\n", " ")   // Для Unix/Linux/macOS
		trimmedCommand = strings.ReplaceAll(trimmedCommand, "\r", " ")   // Для старых Mac (редко)

		trimmedCommand = strings.ReplaceAll(trimmedCommand, "```", "")
		trimmedCommand = strings.ReplaceAll(trimmedCommand, "bash", "")
		trimmedCommand = strings.TrimSpace(trimmedCommand)
		fmt.Print(trimmedCommand)

	} else {
		reThink := regexp.MustCompile(`(?is)<think[^>]*>.*?</think>`)
		cleaned := reThink.ReplaceAllString(correctedCommand, "")
		fmt.Print(" " + cleaned)
	}
}
