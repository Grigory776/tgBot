package telegram

// Функционал бота

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	BotToken = "Брать у FatherBot"
	SuperUser = -1 // id - главного админа для бота
	ApiAssesKey = "Api брать на сайте https://ipstack.com/" // Сервер проверяет ip 
	Limit = 3 // Лимит запросов от одного пользователя
)

type Bot struct{
	bot *tgbotapi.BotAPI
}

func initBot (token string) (*tgbotapi.BotAPI,error){
	bot, err := tgbotapi.NewBotAPI(BotToken)
	if err != nil {
		log.Printf("инициализация бота не выполнена: %v",err)
		return nil, err
	}
	return bot, nil
}

func (b *Bot) Start(){
	var err error
	b.bot, err = initBot(BotToken)
	if err != nil {
		log.Print(err)
	}
	b.bot.Debug = true
	u := tgbotapi.NewUpdate(0)
    u.Timeout = 120
	updts := b.bot.GetUpdatesChan(u)
	for update := range updts{
		if update.Message == nil{
			continue
		}
		err := NewClient(&update) // Регистрируем клиента, если его нету в бд, заносим его туда
		if err != nil {
			panic(err)
		}		
		if update.Message.From.ID == SuperUser{ // Обрабатываем запрос в зависимости от кого он
			b.adminHandler(update)
		} else {
			b.userHandler(update)
		}
	}
}

func (b *Bot) adminHandler(upd tgbotapi.Update){ 
	var Keyboard = tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("рассылка"),
			tgbotapi.NewKeyboardButton("админы"),
			tgbotapi.NewKeyboardButton("история"),
		),
	)
	msg := tgbotapi.NewMessage(upd.Message.Chat.ID,"сообщение не обработано")
	msg.ReplyMarkup = Keyboard
	switch {
	case upd.Message.Text == "/start":
		msg.Text = "Привет, босс"
	case upd.Message.Text== "рассылка":
		msg.Text = "Отправь текст, который хочешь разослать в следующем формате: !!! текст для рассылки"
	case upd.Message.Text[0:3] == "!!!":
		err := b.masMalling(upd.Message.Text[3:])
		if err != nil {
			msg.Text = "сообщение не доставлено"
			log.Printf("сообщение не доставлено:%v",err)
		}else{
			msg.Text = "сообщение доставлено"
		}
	case upd.Message.Text == "история":
		reqs,err := AllRequests()
		if err != nil {
			msg.Text = "проблемы с БД"
			log.Printf("история запросов не получена:%v",err)
		}else {
			for _,val :=range reqs {
				msg.Text = val.String()
				if _, err := b.bot.Send(msg); err != nil {
					log.Printf("сообщение не доставлено: %v",err)
				}
			}
			msg.Text = "."
		}
	case upd.Message.Text == "админы":
		admins,err := GetIdUserAdmin()
		if err != nil {
			msg.Text = "проблемы с БД"
			log.Printf("информация об админах не получена:%v",err)
		} else {
			for _,val :=range admins {
				msg.Text ="@" + val
				if _, err := b.bot.Send(msg); err != nil {
					log.Printf("сообщение не доставлено: %v",err)
				}
			}
			msg.Text = "."
		}
	}
	if _, err := b.bot.Send(msg); err != nil {
		log.Printf("сообщение не доставлено: %v",err)
	}
}

func (b *Bot) masMalling (text string) error{
	chatId,err := GetIdUserExceptAdmin()
	if err != nil {
		return err
	}
	for _,val := range chatId {
		msg := tgbotapi.NewMessage(val,text)
		_,err := b.bot.Send(msg)
		if err != nil{
			return err
		}
	}
	return nil
}

func (b *Bot) userHandler(upd tgbotapi.Update){
	var Keyboard = tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Проверить IP"),
			tgbotapi.NewKeyboardButton("История"),
		),
	)
	var attemps int
	msg := tgbotapi.NewMessage(upd.Message.Chat.ID, "сообщение не обработано")
	msg.ReplyMarkup = Keyboard
	switch {
	case upd.Message.Text == "/start":
		msg.Text = "Привет! Введи IP, которое требуется проверить"
	case attemps >= Limit:
		msg.Text = "Количество попыток закончилось! Попробуй потворить позже!"
	case upd.Message.Text == "История":
		reqs,err := UserRequests(upd.Message.From.ID)
		if err != nil {
			msg.Text = "проблемы с БД"
			log.Printf("история запросов не получена:%v",err)
		}else {
			for _,val :=range reqs {
				msg.Text = val.String()
				if _, err := b.bot.Send(msg); err != nil {
					log.Printf("сообщение не доставлено: %v",err)
				}
			}
			msg.Text = "."
		}
	case upd.Message.Text == "Проверить IP":
		msg.Text = "Напиши IP в следующем формате:\n0-255.0-255.0-255.0-255\nгде 0-255-число из диапозона"
	case IpFormat(upd.Message.Text):
		req, err := RequestIP(upd.Message.Text)
		if err != nil {
			log.Printf("ip не получено:%v",err)
		}
		req.User_id = upd.Message.From.ID
		req.Username = upd.Message.From.UserName
		err = req.InitRequestBaseDate()
		if err != nil {
			log.Printf("пользователь не занесен в БД:%v",err)
		}
		msg.Text = fmt.Sprintf("Данные по ip - %v\nСтрана: %v\nГород: %v\nКоординаты: \n%v  \n%v \nШирина и долгота соответсвенно\n", req.Ip, req.Country, req.Region, req.Latitude, req.Longitude )			
	
}
	if _, err := b.bot.Send(msg); err != nil {
		log.Printf("сообщение не доставлено: %v",err)
	}
}

func IpFormat(str string) bool {
	nm := strings.Split(str,".")
	if len(nm) != 4{
		return false
	}
	for _,val := range nm{
		i,err := strconv.Atoi(val)
		if err != nil{
			return false
		}
		if i > 255 || i < 0{
			return false
		}
	}
	return true
}

func RequestIP (ip string) (*Requests,error){
	url := "http://api.ipstack.com/"+ip+"?access_key="+ApiAssesKey+"&format=1"
	var req *Requests
	var err error
	req,err = UnmarshalURL(url)
	if err != nil {
		return nil, err
	}
	return req, nil 
}

// Функция декодит данные json с url-адреса
func UnmarshalURL (url string) (*Requests,error){
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK{
		return nil, fmt.Errorf("получение %s:%s",url,resp.Status)
	}
	doc,err := io.ReadAll(resp.Body)
	if err != nil {
		return nil,fmt.Errorf("не получилось прочитать:%s",url)
	}
	var req Requests
	if err := json.Unmarshal(doc, &req); err!=nil{
		return nil, fmt.Errorf("не получилось декодировать:%s",url)
	}
	return &req,nil
}

