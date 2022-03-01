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
	"math"
	"os"
	"strings"
	"strconv"
)

// Pass token and sensible APIs through environment variables
const telegramApiBaseUrl string = "https://api.telegram.org/bot"
const telegramApiSendMessage string = "/sendMessage"
const telegramTokenEnv string = "TELEGRAM_BOT_TOKEN"
const telegramApiEditMessage string = "/editMessageText"
const telegramSendLocationMessage string = "/sendLocation"
const ANTON_CHAT_ID int = 49208041

var telegramApiSend string = telegramApiBaseUrl + os.Getenv(telegramTokenEnv) + telegramApiSendMessage
var telegramApiEdit string = telegramApiBaseUrl + os.Getenv(telegramTokenEnv) + telegramApiEditMessage
var telegramApiSendLocation string = telegramApiBaseUrl + os.Getenv(telegramTokenEnv) + telegramSendLocationMessage

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

// haversin(θ) function
func hsin(theta float64) float64 {
	return math.Pow(math.Sin(theta/2), 2)
}

// Distance function returns the distance (in meters) between two points of
//     a given longitude and latitude relatively accurately (using a spherical
//     approximation of the Earth) through the Haversin Distance Formula for
//     great arc distance on a sphere with accuracy for small distances
//
// point coordinates are supplied in degrees and converted into rad. in the func
//
// distance returned is METERS!!!!!!
// http://en.wikipedia.org/wiki/Haversine_formula
func Distance(l1, l2 Location) float64 {
	// convert to radians
  // must cast radius as float to multiply later
	var la1, lo1, la2, lo2, r float64
	la1 = l1.Latitude * math.Pi / 180
	lo1 = l1.Longitude * math.Pi / 180
	la2 = l2.Latitude * math.Pi / 180
	lo2 = l2.Longitude * math.Pi / 180

	r = 6378100 // Earth radius in METERS

	// calculate
	h := hsin(la2-la1) + math.Cos(la1)*math.Cos(la2)*hsin(lo2-lo1)

	return 2 * r * math.Asin(math.Sqrt(h))
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
	Location Location `json:"location"`
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
	Username string `json:"username"`
}

type Location struct {
	Longitude float64 `json:"longitude"`
	Latitude  float64 `json:"latitude"`
}

// Implements the fmt.String interface to get the representation of a Chat as a string.
func (c Chat) String() string {
	return fmt.Sprintf("(id: %d)", c.Id)
}

var ALLOWED_USERS = [...]string {"antonhulikau", "sonicfelidae"}

var LOCATIONS = [...]Location {
	Location {Latitude: 48.158967, Longitude: 11.490981}, // nyphemburg
	Location {Latitude: 48.155582, Longitude: 11.493340}, // west
	Location {Latitude: 48.143296, Longitude: 11.596526}, // ducks
	Location {Latitude: 48.173194, Longitude: 11.555078}, // olympia
	Location {Latitude: 48.166302, Longitude: 11.568141}, // luitpold
}

func isAllowed(e string) bool {
    for _, a := range ALLOWED_USERS {
        if a == e {
            return true
        }
    }
    return false
}

func to_lower_letters(s string) string {
	ret := strings.ToLower(s);
	return ret;
}

// HandleTelegramWebHook sends a message back to the chat with a punchline starting by the message provided by the user.
func HandleTelegramWebHook(w http.ResponseWriter, r *http.Request) {

	// Parse incoming request
	var update, err = parseTelegramRequest(r)
	if err != nil {
		log.Printf("error parsing update, %s", err.Error())
		return
	}

	if (!isAllowed(update.Message.Chat.Username)) {
		return;
	}

	if (update.Message.Text == "/start") {
		var telegramResponseBody, errTelegram = sendTextMessage(update.Message.Chat.Id, "Присылай мне свою локацию. Если ты будешь относительно близко к расположению подсказки, я дам тебе точные координаты!\nУ меня есть так же команда /unlock =)")
		sendTextMessage(ANTON_CHAT_ID, "Соня начала искать локации!")
		if errTelegram != nil {
			log.Printf("got error %s from telegram, response body is %s", errTelegram.Error(), telegramResponseBody)
		} else {
			log.Printf("successfully distributed to chat id %d", update.Message.Chat.Id)
		}
	} else if (update.Message.Text == "/unlock") {
		var telegramResponseBody, errTelegram = sendTextMessage(update.Message.Chat.Id, "Пароль?")
		if errTelegram != nil {
			log.Printf("got error %s from telegram, response body is %s", errTelegram.Error(), telegramResponseBody)
		} else {
			log.Printf("successfully distributed to chat id %d", update.Message.Chat.Id)
		}
	} else if (to_lower_letters(update.Message.Text) == "afsio") {
		var telegramResponseBody, errTelegram = sendTextMessage(update.Message.Chat.Id, "Молодец! Все верно!\nВ качестве приза могли прийти, но не пришли:\n1. Поездка в Австрию на викенд. Но она почему-то вводит локдаун.\n2. Поход на Щелкунчика. Но кто-то прощелкал все полимеры =(.\n3. Карты с покемонами на испанском. Но они у тебя уже есть.\n\n\n\nНо зато пришел: бессрочный recharge day on demand. Предложение отвезти тебя, куда ты захочешь, на 1 день. Используй его, когда тебе вздумается.")
		var telegramResponseBody2, errTelegram2 = sendTextMessage(ANTON_CHAT_ID, "Соня справилась!")
		if errTelegram != nil {
			log.Printf("got error %s from telegram, response body is %s", errTelegram.Error(), telegramResponseBody)
		} else {
			log.Printf("successfully distributed to chat id %d", update.Message.Chat.Id)
		}
		if errTelegram2 != nil {
			log.Printf("got error %s from telegram, response body is %s", errTelegram2.Error(), telegramResponseBody2)
		} else {
			log.Printf("successfully distributed to chat id %d", update.Message.Chat.Id)
		}
	} else if (update.Message.Location.Latitude > 0) {
		found := false;
		for t, l := range LOCATIONS {
			if (Distance(l, update.Message.Location) < 2000) {
				var telegramResponseBody, errTelegram = sendTextMessage(update.Message.Chat.Id, "Проверь это место")
				sendTextMessage(ANTON_CHAT_ID, fmt.Sprintf("Соня проверяет %d!", t))
				if errTelegram != nil {
					log.Printf("got error %s from telegram, response body is %s", errTelegram.Error(), telegramResponseBody)
				} else {
					log.Printf("successfully distributed to chat id %d", update.Message.Chat.Id)
				}
				telegramResponseBody, errTelegram = sendLocationMessage(update.Message.Chat.Id, l)
				if errTelegram != nil {
					log.Printf("got error %s from telegram, response body is %s", errTelegram.Error(), telegramResponseBody)
				} else {
					log.Printf("successfully distributed to chat id %d", update.Message.Chat.Id)
				}
				found = true;
			}
		}
		if (!found) {
			var telegramResponseBody, errTelegram = sendTextMessage(update.Message.Chat.Id, "Вблизи нет подсказок")
			if errTelegram != nil {
				log.Printf("got error %s from telegram, response body is %s", errTelegram.Error(), telegramResponseBody)
			} else {
				log.Printf("successfully distributed to chat id %d", update.Message.Chat.Id)
			}
		}
	} else {
		var telegramResponseBody, errTelegram = sendTextMessage(update.Message.Chat.Id, "Этот пароль не подходит =(")
		sendTextMessage(ANTON_CHAT_ID, fmt.Sprintf("Соня ввела %s!", update.Message.Text))
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
func sendTextMessage(chatId int, text string) (string, error) {
	log.Printf("Sending start message to chat_id: %d", chatId);

	response, err := http.PostForm(
		telegramApiSend,
		url.Values{
			"chat_id": {strconv.Itoa(chatId)},
			"text": {text},
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

func sendLocationMessage(chatId int, l Location) (string, error) {
	log.Printf("Sending location message to chat_id: %d", chatId);

	response, err := http.PostForm(
		telegramApiSendLocation,
		url.Values{
			"chat_id": {strconv.Itoa(chatId)},
			"longitude": {strconv.FormatFloat(l.Longitude, 'E', -1, 64)},
			"latitude": {strconv.FormatFloat(l.Latitude, 'E', -1, 64)},
			"horizontal_accuracy": {"2"},
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