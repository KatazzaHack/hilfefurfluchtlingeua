// Package handler contains an HTTP Cloud Function to handle update from Telegram whenever a users interacts with the
// bot.
package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
)

// Pass token and sensible APIs through environment variables
const telegramApiBaseUrl string = "https://api.telegram.org/bot"
const telegramApiSendMessage string = "/sendMessage"
const telegramTokenEnv string = "TELEGRAM_BOT_TOKEN"
const telegramApiEditMessage string = "/editMessageText"

var telegramApiSend string = telegramApiBaseUrl + os.Getenv(telegramTokenEnv) + telegramApiSendMessage
var telegramApiEdit string = telegramApiBaseUrl + os.Getenv(telegramTokenEnv) + telegramApiEditMessage

// Update is a Telegram object that we receive every time an user interacts with the bot.
type Update struct {
	UpdateId int     `json:"update_id"`
	Message  Message `json:"message"`
	CallbackQuerry CallbackQuerry `json:"callback_query"`
}

// Implements the fmt.String interface to get the representation of an Update as a string.
func (u Update) String() string {
	return fmt.Sprintf("(update id: %d, message: %s, callback: %s)", u.UpdateId, u.Message, u.CallbackQuerry)
}

// Message is a Telegram object that can be found in an update.
// Note that not all Update contains a Message. Update for an Inline Query doesn't.
type Message struct {
	Id       int      `json:"message_id"`
	Text     string   `json:"text"`
	Chat     Chat     `json:"chat"`
	Audio    Audio    `json:"audio"`
	Voice    Voice    `json:"voice"`
	Document Document `json:"document"`
}

type CallbackQuerry struct {
	Id string `json:"id"`
	From User `json:"from"`
	Data string `json:"data"`
	Message Message `json:"message"`
	InlineMessageId string `json:"inline_message_id"`
}

func (c CallbackQuerry) String() string {
	return fmt.Sprintf("(id: %d, message: %s, data: %s, from: %s)", c.Id, c.Message, c.Data, c.From)
}

type User struct {
	Id int64 `json:"id"`
	Username string `json:"username"`
}

// Implements the fmt.String interface to get the representation of a Message as a string.
func (m Message) String() string {
	return fmt.Sprintf("(text: %s, chat: %s, audio %s)", m.Text, m.Chat, m.Audio)
}

// Audio message has extra attributes
type Audio struct {
	FileId   string `json:"file_id"`
	Duration int    `json:"duration"`
}

// Implements the fmt.String interface to get the representation of an Audio as a string.
func (a Audio) String() string {
	return fmt.Sprintf("(file id: %s, duration: %d)", a.FileId, a.Duration)
}

// Voice Message can be summarized with similar attribute as an Audio message for our use case.
type Voice Audio

// Document Message refer to a file sent.
type Document struct {
	FileId   string `json:"file_id"`
	FileName string `json:"file_name"`
}

// Implements the fmt.String interface to get the representation of an Document as a string.
func (d Document) String() string {
	return fmt.Sprintf("(file id: %s, file name: %s)", d.FileId, d.FileName)
}

// A Chat indicates the conversation to which the Message belongs.
type Chat struct {
	Id int `json:"id"`
}

// Implements the fmt.String interface to get the representation of a Chat as a string.
func (c Chat) String() string {
	return fmt.Sprintf("(id: %d)", c.Id)
}

var CELEBRATIONS = [...]string {
"Твой друг: Дрюня\nНа вопрос: Что бы ты приготовил/а Маше на завтрак?\nОтветил(а): Пельмеши",
}

var ALLOWED_USERS = [...]string {"antonhulikau", "okalitova", "maffina95"}

func isAllowed(e string) bool {
    for _, a := range ALLOWED_USERS {
        if a == e {
            return true
        }
    }
    return false
}

// HandleTelegramWebHook sends a message back to the chat with a punchline starting by the message provided by the user.
func HandleTelegramWebHook(w http.ResponseWriter, r *http.Request) {

	// Parse incoming request
	var update, err = parseTelegramRequest(r)
	if err != nil {
		log.Printf("error parsing update, %s", err.Error())
		return
	}

	if (update.Message.Text == "/start") {
		var telegramResponseBody, errTelegram = sendStartTextMessage(update.Message.Chat.Id, "Привет, нажимай на кнопку получить поздравление и кайфуй!")
		if errTelegram != nil {
			log.Printf("got error %s from telegram, response body is %s", errTelegram.Error(), telegramResponseBody)
		} else {
			log.Printf("successfully distributed to chat id %d", update.Message.Chat.Id)
		}
	} else if (isAllowed(update.CallbackQuerry.From.Username)) {
		p, _ := strconv.Atoi(update.CallbackQuerry.Data);

		var telegramResponseBody, errTelegram = sendCelebrateMessage(update.CallbackQuerry.Message.Chat.Id, update.CallbackQuerry.Message.Id, p);
		if errTelegram != nil {
			log.Printf("got error %s from telegram, response body is %s", errTelegram.Error(), telegramResponseBody)
		} else {
			log.Printf("successfully distributed to chat id %d", update.Message.Chat.Id)
		}
	}
	log.Printf("Update new is %s", update);
}

// parseTelegramRequest handles incoming update from the Telegram web hook
func parseTelegramRequest(r *http.Request) (*Update, error) {
	var update Update
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		log.Printf("could not decode incoming update %s", err.Error())
		return nil, err
	}
	if update.UpdateId == 0 {
		log.Printf("invalid update id, got update id = 0")
		return nil, errors.New("invalid update id of 0 indicates failure to parse incoming update")
	}
	return &update, nil
}

// sendTextToTelegramChat sends an initial text message to the Telegram chat identified by its chat Id
func sendStartTextMessage(chatId int, text string) (string, error) {
	log.Printf("Sending start message to chat_id: %d", chatId);

	keyboard := make(map[string][][]map[string]string)
	var fo = []map[string]string{}
	fo = append(fo, map[string]string {"text": "Получить поздравление", "callback_data": "0"})
	keyboard["inline_keyboard"] = [][]map[string]string{fo}

    keyboardStr, err := json.Marshal(keyboard)
	response, err := http.PostForm(
		telegramApiSend,
		url.Values{
			"chat_id": {strconv.Itoa(chatId)},
			"text": {text},
			"reply_markup": {string(keyboardStr)},
		},
	)
	if err != nil {
		log.Printf("error when posting text to the chat: %s", err.Error())
		return "", err
	}
	defer response.Body.Close()
	var bodyBytes, errRead = ioutil.ReadAll(response.Body)
	if errRead != nil {
		log.Printf("error in parsing telegram answer %s", errRead.Error())
		return "", err
	}
	bodyString := string(bodyBytes)
	log.Printf("Body of Telegram Response: %s", bodyString)

	return bodyString, nil
}

// sendTextToTelegramChat sends an initial text message to the Telegram chat identified by its chat Id
func sendCelebrateMessage(chatId int, messageId int, p int) (string, error) {
	log.Printf("Sending start message to chat_id: %d", chatId);

	text := CELEBRATIONS[p];
	p += 1;
	if (p == len(CELEBRATIONS)) {
		p = 0;
	}

	keyboard := make(map[string][][]map[string]string)
	var fo = []map[string]string{}
	fo = append(fo, map[string]string {"text": "Получить поздравление", "callback_data": strconv.Itoa(p)})
	keyboard["inline_keyboard"] = [][]map[string]string{fo}

    keyboardStr, err := json.Marshal(keyboard)
	response, err := http.PostForm(
		telegramApiEdit,
		url.Values{
			"chat_id": {strconv.Itoa(chatId)},
			"message_id": {strconv.Itoa(messageId)},
			"text": {text},
			"reply_markup": {string(keyboardStr)},
		},
	)
	if err != nil {
		log.Printf("error when posting text to the chat: %s", err.Error())
		return "", err
	}
	defer response.Body.Close()
	var bodyBytes, errRead = ioutil.ReadAll(response.Body)
	if errRead != nil {
		log.Printf("error in parsing telegram answer %s", errRead.Error())
		return "", err
	}
	bodyString := string(bodyBytes)
	log.Printf("Body of Telegram Response: %s", bodyString)

	return bodyString, nil
}