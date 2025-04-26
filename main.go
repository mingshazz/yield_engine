package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// Data model
type Data struct {
	ID     uint    `gorm:"primaryKey" json:"id"`
	UNIX   int64   `json:"unix"`
	SYMBOL string  `json:"symbol"`
	OPEN   float64 `json:"open"`
	HIGH   float64 `json:"high"`
	LOW    float64 `json:"low"`
	CLOSE  float64 `json:"close"`
}

var db *gorm.DB

func initDB() {
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?parseTime=true",
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASS"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_NAME"),
	)
	var err error
	db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("Failed to connect to database: " + err.Error())
	}
	db.AutoMigrate(&Data{})
}

func main() {
	initDB()

	r := gin.Default()
	r.MaxMultipartMemory = 8 << 30 // 8 GiB file limit

	r.POST("/data", uploadData)
	r.GET("/data", getData)

	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Endpoint not found"})
	})

	r.Run(":8080")
}

// POST /data
func uploadData(c *gin.Context) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File upload error"})
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not open uploaded file"})
		return
	}
	defer file.Close()

	reader := csv.NewReader(file)

	// Validate header
	header, err := reader.Read()
	if err != nil || !validateHeader(header) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid or missing CSV header"})
		return
	}

	const batchSize = 1000
	var batch []Data
	totalInserted := 0

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}

		data, err := parseRecord(record)
		if err == nil {
			batch = append(batch, data)
		}

		if len(batch) >= batchSize {
			if err := db.Create(&batch).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to insert batch into DB"})
				return
			}
			totalInserted += len(batch)
			batch = batch[:0]
		}
	}

	if len(batch) > 0 {
		if err := db.Create(&batch).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to insert final batch into DB"})
			return
		}
		totalInserted += len(batch)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Data uploaded successfully", "inserted_records": totalInserted})
}

// GET /data
func getData(c *gin.Context) {
	var results []Data
	query := db.Model(&Data{})

	symbol := c.Query("symbol")
	if symbol != "" {
		query = query.Where("symbol LIKE ?", "%"+symbol+"%")
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	offset := (page - 1) * limit

	var total int64
	query.Count(&total)

	query.Limit(limit).Offset(offset).Find(&results)

	c.JSON(http.StatusOK, gin.H{
		"total":   total,
		"page":    page,
		"limit":   limit,
		"results": results,
	})
}

// Header validator
func validateHeader(header []string) bool {
	expected := []string{"UNIX", "SYMBOL", "OPEN", "HIGH", "LOW", "CLOSE"}
	if len(header) != len(expected) {
		return false
	}
	for i := range header {
		if strings.TrimSpace(strings.ToUpper(header[i])) != expected[i] {
			return false
		}
	}
	return true
}

// Convert CSV row into struct
func parseRecord(record []string) (Data, error) {
	if len(record) != 6 {
		return Data{}, fmt.Errorf("invalid record length")
	}
	unix, err1 := strconv.ParseInt(record[0], 10, 64)
	open, err2 := strconv.ParseFloat(record[2], 64)
	high, err3 := strconv.ParseFloat(record[3], 64)
	low, err4 := strconv.ParseFloat(record[4], 64)
	closeVal, err5 := strconv.ParseFloat(record[5], 64)

	if err1 != nil || err2 != nil || err3 != nil || err4 != nil || err5 != nil {
		return Data{}, fmt.Errorf("parsing error")
	}

	return Data{
		UNIX:   unix,
		SYMBOL: record[1],
		OPEN:   open,
		HIGH:   high,
		LOW:    low,
		CLOSE:  closeVal,
	}, nil
}
