package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"sync"
)

// Config структура для хранения конфигурационных данных
type Config struct {
	Token       string `yaml:"token"`
	SaveFormURL string `yaml:"saveFormURL"`
	GetFormsURL string `yaml:"getFormsURL"`
}

// Глобальная переменная для хранения конфигурации
var config Config

// Структура Form для создание формы для пост запроса
type Form struct {
	PeriodStart         string
	PeriodEnd           string
	PeriodKey           string
	IndicatorToMoID     int
	IndicatorToMoFactID int
	Value               int
	FactTime            string
	IsPlan              int
	AuthUserID          int
	Comment             string
}

func init() {
	// Чтение файла конфигурации
	data, err := ioutil.ReadFile("config.yaml")
	if err != nil {
		log.Fatalf("Error reading config file: %v", err)
	}

	// Разбор данных YAML
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		log.Fatalf("Error parsing config file: %v", err)
	}
}

// Проверка факта отправки данных
func saveData(data Form) (bool, error) {
	form := url.Values{} // создание формы и ее заполнение
	form.Add("period_start", data.PeriodStart)
	form.Add("period_end", data.PeriodEnd)
	form.Add("period_key", data.PeriodKey)
	form.Add("indicator_to_mo_id", fmt.Sprintf("%d", data.IndicatorToMoID))
	form.Add("indicator_to_mo_fact_id", fmt.Sprintf("%d", data.IndicatorToMoFactID))
	form.Add("value", fmt.Sprintf("%d", data.Value))
	form.Add("fact_time", data.FactTime)
	form.Add("is_plan", fmt.Sprintf("%d", data.IsPlan))
	form.Add("auth_user_id", fmt.Sprintf("%d", data.AuthUserID))
	form.Add("comment", data.Comment)

	req, err := http.NewRequest(http.MethodPost, config.SaveFormURL, bytes.NewBufferString(form.Encode())) // создание пост запроса
	if err != nil {
		fmt.Println("Error creating request:", err)
		return false, err
	}
	req.Header.Add("Authorization", "Bearer "+config.Token)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}    // создание клиента для запроса
	resp, err := client.Do(req) // запрос
	if err != nil {
		fmt.Println("Error making request:", err)
		return false, err
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK, nil // возращаем true если ответ 200
}

// Запрос сохраненных данных с сайта
func getData(data Form) map[string]interface{} {
	form := url.Values{} // создание формы и ее заполнение
	form.Add("period_start", data.PeriodStart)
	form.Add("period_end", data.PeriodEnd)
	form.Add("period_key", data.PeriodKey)
	form.Add("indicator_to_mo_id", strconv.Itoa(data.IndicatorToMoID))

	req, err := http.NewRequest("POST", config.GetFormsURL, bytes.NewBufferString(form.Encode())) // создание пост запроса
	if err != nil {
		fmt.Println("Error creating request:", err)
		return nil
	}
	req.Header.Add("Authorization", "Bearer "+config.Token)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}    // создание клиента для запроса
	resp, err := client.Do(req) // запрос
	if err != nil {
		fmt.Println("Error making request:", err)
		return nil
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body) // после разбираем ответ на json
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return nil
	}
	var result map[string]interface{}

	if err := json.Unmarshal(body, &result); err != nil { // пытаемся записать json в ответ
		fmt.Println("Error unmarshalling response:", err)
		return nil
	}

	return result
}

func main() {

	var buffer []Form
	var checkData []string
	for i := 0; i < 10; i++ { // создание 10 запросов в БД
		number := strconv.Itoa(rand.Intn(1024))
		checkData = append(checkData, number)
		buffer = append(buffer, Form{
			PeriodStart:         "2024-05-01",
			PeriodEnd:           "2024-05-31",
			PeriodKey:           "month",
			IndicatorToMoID:     227373,
			IndicatorToMoFactID: 0,
			Value:               1,
			FactTime:            "2024-05-31",
			IsPlan:              0,
			AuthUserID:          40,
			Comment:             "buffer Levchikov" + number,
		})
	}

	wg := sync.WaitGroup{} // создание WaitGroup
	mx := sync.Mutex{}     // так как нам нужно чтобы 1 запрос прошел весь цикл создаем mutex
	wg.Add(1)

	go func(buffer []Form) { // горутина для прохода по всем запросам
		defer wg.Done()
		for _, value := range buffer {
			mx.Lock() // блокируем mutex для запроса
			ok, err := saveData(value)
			if err != nil {
				log.Fatal(err)
			}
			if ok { // если ответ 200 то выводим сообщение о том что данные записаны 
				fmt.Println("Record saved successfully:", value)
			}
			mx.Unlock() // разблокируем mutex
		}
	}(buffer)

	wg.Wait() // так как не знаем сколько будут работать наша горутина, создали wg
	// и только после того как отработает горутина main закончиться

	result := getData(buffer[0]) // сохраненные данных с сайта

	data := result["DATA"].(map[string]interface{}) // обходим json чтобы дойти до записей
	rows := data["rows"].([]interface{})

	cnt := 0

	for _, row := range rows {
		comment := row.(map[string]interface{})["comment"] // Ищем свои записи по комментариям

		if comment == "buffer Levchikov"+checkData[len(checkData)-cnt-1] { // добавил случайное число чтобы точно знать что все записи записаны
			cnt++
		}
		if cnt == len(buffer) {
			fmt.Println("All entries are successfully saved.")
			break
		}
	}
}
