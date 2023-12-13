package main

import (
	"flag"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"log"
	"net"
	"os"
)

var db *gorm.DB
var resetDB bool

func init() {
	// Регистрация флага
	flag.BoolVar(&resetDB, "resetdb", false, "Пересоздать базу данных")
	flag.Parse()
}

type Config struct {
	Database struct {
		Name string `yaml:"name"`
	} `yaml:"database"`
	Server struct {
		Port string `yaml:"port"`
	} `yaml:"server"`
}

// Server модель для базы данных
type Server struct {
	ID         uint `gorm:"primary_key"`
	Name       string
	IPAddress  string
	IP6Address string
	Location   string
	Hoster     string
	Comment    string
}

// loadConfig загружает конфигурацию из YAML-файла
func loadConfig(filename string) (*Config, error) {
	config := &Config{}
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(data, config)
	if err != nil {
		return nil, err
	}
	return config, nil
}

func main() {

	// Load config
	config, err := loadConfig("config.yaml")
	if err != nil {
		log.Fatal("Error loading configuration:", err)
	}

	// Проверка флага resetDB и пересоздание базы данных при необходимости
	if resetDB {
		resetDatabase(config.Database.Name)
		return
	}

	// Подключение к базе данных SQLite
	db, err = gorm.Open("sqlite3", config.Database.Name)
	if err != nil {
		log.Fatal("Ошибка подключения к базе данных:", err)
	}
	defer db.Close()
	db.AutoMigrate(&Server{})

	// Проверка наличия таблицы, и создание, если ее нет
	if !db.HasTable(&Server{}) {
		if err := db.CreateTable(&Server{}).Error; err != nil {
			log.Fatal("Ошибка создания таблицы:", err)
		}
	}

	// Инициализация маршрутизатора Gin
	r := gin.Default()
	r.LoadHTMLGlob("templates/*.html")

	// Маршруты
	r.GET("/", func(c *gin.Context) {
		var servers []Server
		db.Find(&servers)
		fmt.Println("Сервера из базы данных:", servers)
		c.HTML(200, "index.html", gin.H{"servers": servers})
	})

	r.GET("/add_server", func(c *gin.Context) {
		c.HTML(200, "add_server.html", nil)
	})

	r.POST("/add_server", func(c *gin.Context) {
		var server Server
		c.Bind(&server)
		db.Create(&server)
		c.Redirect(302, "/")
	})

	r.GET("/edit_server/:id", func(c *gin.Context) {
		var server Server
		id := c.Param("id")
		db.First(&server, id)
		c.HTML(200, "edit_server.html", gin.H{"server": server})
	})

	r.POST("/edit_server/:id", func(c *gin.Context) {
		var server Server
		id := c.Param("id")
		db.First(&server, id)
		c.Bind(&server)
		db.Save(&server)
		c.Redirect(302, "/")
	})

	r.GET("/delete_server/:id", func(c *gin.Context) {
		id := c.Param("id")
		var server Server
		db.First(&server, id)
		db.Delete(&server)
		c.Redirect(302, "/")
	})

	// ...

	// Маршрут для экспорта данных в CSV
	r.GET("/export", func(c *gin.Context) {
		var servers []Server
		db.Find(&servers)

		if len(servers) == 0 {
			c.String(200, "No data to export")
			return
		}

		// Формирование CSV
		csvData := "Name,IP Address, IPv6 Address,Location,Hoster,Comment\n"
		for _, server := range servers {
			// Заключаем каждое поле в двойные кавычки
			csvData += fmt.Sprintf("\"%s\",\"%s\",\"%s\",\"%s\",\"%s\",\"%s\"\n", server.Name, server.IPAddress, server.IP6Address, server.Location, server.Hoster, server.Comment)
		}

		// Отправка CSV клиенту
		c.Header("Content-Disposition", "attachment;filename=servers.csv")
		c.Data(200, "text/csv", []byte(csvData))
	})

	// Получение IP-адреса и порта сервера
	host, port, err := net.SplitHostPort(os.Getenv("HOST"))
	if err != nil {
		host = "localhost"
		port = config.Server.Port
	}

	// Вывод IP-адреса и порта в консоль
	fmt.Printf("Server is running at http://%s:%s\n", host, port)

	// Запуск веб-сервера
	err = r.Run(fmt.Sprintf("%s:%s", host, port))
	if err != nil {
		log.Fatal(err)
	}
}

// Функция для пересоздания базы данных
// Функция для пересоздания базы данных
func resetDatabase(dbName string) {
	fmt.Println("Пересоздание базы данных...")

	// Delete file
	err := os.Remove(dbName)
	if err != nil {
		log.Fatal(err)
	}

	// Повторное открытие соединения с базой данных
	db, err := gorm.Open("sqlite3", "servers.db")
	if err != nil {
		log.Fatal("Ошибка при повторном открытии базы данных:", err)
	}

	// Попытка создания таблицы
	if err := db.AutoMigrate(&Server{}).Error; err != nil {
		// Если произошла ошибка, печатаем предупреждение, но не завершаем программу
		fmt.Println("Предупреждение: таблица уже существует:", err)
	}

	// Инициализация данных
	serverData := []Server{
		{Name: "Server1", IPAddress: "192.168.1.1", IP6Address: "N/A", Location: "Office A", Hoster: "Hosting A", Comment: "KZ"},
		{Name: "Server2", IPAddress: "192.168.1.2", IP6Address: "N/A", Location: "Office B", Hoster: "Hosting B", Comment: "EU"},
		{Name: "Server3", IPAddress: "192.168.1.3", IP6Address: "N/A", Location: "Central DC", Hoster: "Hosting C", Comment: "CIS"},
	}

	for _, server := range serverData {
		if err := db.Create(&server).Error; err != nil {
			log.Fatal("Ошибка при добавлении данных:", err)
		}
	}

	fmt.Println("База данных успешно пересоздана.")
}
