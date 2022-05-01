package telegram

// Функционал по работе с БД
import (
	"log"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)


const dsn = "Данные по базе данных (host, user ...)"

// Структура пользователей для БД
type Users struct{ 
	User_id int64
	Chat_id int64 
	First_name string 
	Last_name string 
	Username string 
	Language_code string 
	Admin bool
}

func OpenDB(inf_db string) (*gorm.DB, error){
	db, err := gorm.Open(postgres.Open(inf_db), &gorm.Config{})
	if err != nil {
		log.Printf("не удалось открыть базу данных: %V",err)
		return nil, err
	}
	return db,nil
}

func (us *Users) InitUserBaseDates()error{
	db, err := OpenDB(dsn)
	if err != nil {
		return err
	}
	em, err := us.PresenceUserBaseDates();
	if err != nil {
		return err
	}
	if em {
		log.Print("Пользователь уже есть в БД\n\n\n\n\n\n")
		return nil
	}
	db.Create(us)
	return nil;
}

func (us *Users) PresenceUserBaseDates()(bool,error){
	db, err := OpenDB(dsn)
	if err != nil {
		return false, err
	}
	var res bool
	db.First(us, us.User_id).Scan(&res)
	return res, nil 
}

func (us *Users) GetChatID() (int64, error){
	db, err := OpenDB(dsn)
	if err != nil {
		return -1, err
	}
	var res int64
	db.First(us, us.Chat_id).Scan(&res)
	return res, nil
} 

// Регистрация нового клиента в БД, елси его там нет, по обновлению в телеграмм
func NewClient(upd *tgbotapi.Update) error{
	var us = Users{
		User_id: upd.Message.From.ID,
		Chat_id: upd.Message.Chat.ID,
		First_name: upd.Message.From.FirstName,
		Last_name: upd.Message.From.LastName,
		Username: upd.Message.From.UserName,
		Language_code: upd.Message.From.LanguageCode,
	}
	if us.User_id == SuperUser{
		us.Admin = true
	} 
	err := us.InitUserBaseDates()
	if err != nil {
		return err
	}
	return nil
}

// Функция возвращает ID людей, не являющихся админами
func GetIdUserExceptAdmin() ([]int64,error) {
	db, err := OpenDB(dsn)
	if err != nil {
		return nil, err
	}
	var res []int64
	var user Users
	db.Select("chat_id").Where("admin = ?",false).Find(&user).Scan(&res)
	return res, nil
}

// Функция возвращает username людей, являющихся админами
func GetIdUserAdmin() ([]string,error) {
	db, err := OpenDB(dsn)
	if err != nil {
		return nil, err
	}
	var res []string
	var user Users
	db.Select("username").Where("admin = ?",true).Find(&user).Scan(&res)
	return res, nil
}


// Структура запросов для БД
type Requests struct{
	User_id int64
	Username string
	Country string `json:"country_name"`
	Region string `json:"region_name"`
	Latitude float64 `json:"latitude"` // Ширина
	Longitude float64 `json:"longitude"`
	Ip string `json:"ip"`
}

func (r Requests) String() string{
	return fmt.Sprintf("id: %v\nusrname: @%v\nЗапрос\nIP: %v\nСтрана: %v\nРегион: %v\nШирина: %v\nДолгота: %v\n", r.User_id, r.Username, r.Ip, r.Country,r.Region,r.Latitude,r.Longitude)
}

// Функция возвращает из базы данных все запросы
func AllRequests() ([]Requests, error){
	db, err := OpenDB(dsn)
	if err != nil {
		return nil, err
	}
	var request Requests
	var res []Requests
	db.Find(&request).Scan(&res)
	return res, nil 
}

// Функция возвращает из базы данных запросы определенного юзера
func UserRequests(idUser int64)([]Requests, error){
	db, err := OpenDB(dsn)
	if err != nil {
		return nil, err
	}
	var req Requests
	var res []Requests
	db.Find(&req).Where("user_id = ?",idUser).Scan(&res)
	return res, nil 
}

func (req *Requests)InitRequestBaseDate()error{
		db, err := OpenDB(dsn)
		if err != nil {
			return err
		}
		db.Create(req)
		return nil;
}
