package main

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
)

var db *sql.DB

type Record struct {
	Unix   int64   `json:"unix"`
	Symbol string  `json:"symbol"`
	Open   float64 `json:"open"`
	High   float64 `json:"high"`
	Low    float64 `json:"low"`
	Close  float64 `json:"close"`
}

func initDB() {
	user := os.Getenv("DB_USER")
	pass := os.Getenv("DB_PASS")
	host := os.Getenv("DB_HOST")
	dbname := os.Getenv("DB_NAME")

	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?parseTime=true", user, pass, host, dbname)

	var err error
	db, err = sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("failed to connect to DB: %v", err)
	}

	err = db.Ping()
	if err != nil {
		log.Fatalf("failed to initialize database: %v", err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS records (
			id INT AUTO_INCREMENT PRIMARY KEY,
			unix BIGINT,
			symbol VARCHAR(20),
			open DOUBLE,
			high DOUBLE,
			low DOUBLE,
			close DOUBLE
		);
	`)
	if err != nil {
		log.Fatalf("failed to create table: %v", err)
	}
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: .env file not found. Make sure environment variables are set.")
	}

	initDB()

	router := gin.Default()

	router.POST("/data", uploadCSV)
	router.GET("/data", getData)

	log.Println("Server running on :8080")
	router.Run(":8080")
}

func uploadCSV(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}

	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to open file"})
		return
	}
	defer src.Close()

	reader := csv.NewReader(src)
	reader.FieldsPerRecord = -1
	reader.ReuseRecord = true

	header, err := reader.Read()
	if err != nil || len(header) < 6 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid CSV header"})
		return
	}

	expected := []string{"UNIX", "SYMBOL", "OPEN", "HIGH", "LOW", "CLOSE"}
	for i, col := range expected {
		if strings.TrimSpace(header[i]) != col {
			c.JSON(http.StatusBadRequest, gin.H{"error": "CSV header must match: UNIX,SYMBOL,OPEN,HIGH,LOW,CLOSE"})
			return
		}
	}

	stmt, err := db.Prepare("INSERT INTO records (unix, symbol, open, high, low, close) VALUES (?, ?, ?, ?, ?, ?)")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to prepare insert"})
		return
	}
	defer stmt.Close()

	inserted := 0
	for {
		row, err := reader.Read()
		if err != nil {
			break
		}

		if len(row) < 6 {
			log.Println("Skipping short row:", row)
			continue
		}

		unix, err1 := strconv.ParseInt(row[0], 10, 64)
		open, err2 := strconv.ParseFloat(row[2], 64)
		high, err3 := strconv.ParseFloat(row[3], 64)
		low, err4 := strconv.ParseFloat(row[4], 64)
		close, err5 := strconv.ParseFloat(row[5], 64)

		if err1 != nil || err2 != nil || err3 != nil || err4 != nil || err5 != nil {
			log.Println("Skipping malformed row:", row)
			continue
		}

		_, err = stmt.Exec(unix, row[1], open, high, low, close)
		if err != nil {
			log.Printf("Insert error: %v\n", err)
			continue
		}

		inserted++
	}

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Uploaded %d records", inserted),
	})
}

func getData(c *gin.Context) {
	symbol := c.Query("symbol")
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "10")

	page, err1 := strconv.Atoi(pageStr)
	limit, err2 := strconv.Atoi(limitStr)

	if err1 != nil || err2 != nil || page < 1 || limit < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid pagination parameters"})
		return
	}

	offset := (page - 1) * limit

	query := "SELECT unix, symbol, open, high, low, close FROM records"
	args := []interface{}{}

	if symbol != "" {
		query += " WHERE symbol = ?"
		args = append(args, symbol)
	}

	query += " ORDER BY unix LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := db.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query database"})
		return
	}
	defer rows.Close()

	var records []Record
	for rows.Next() {
		var r Record
		err := rows.Scan(&r.Unix, &r.Symbol, &r.Open, &r.High, &r.Low, &r.Close)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read data"})
			return
		}
		records = append(records, r)
	}

	c.JSON(http.StatusOK, gin.H{
		"page":    page,
		"limit":   limit,
		"results": records,
	})
}
