package mylog

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"
)

// 独立的错误日志记录函数
func ErrLogToFile(level, message string) {
	filename := time.Now().Format("2006-01-02") + "-error.log"
	filepath := logPath + "/" + filename

	file, err := os.OpenFile(filepath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Error opening log file:", err)
		return
	}
	defer file.Close()

	logEntry := fmt.Sprintf("[%s] %s: %s\n", time.Now().Format("2006-01-02T15:04:05"), level, message)
	if _, err := file.WriteString(logEntry); err != nil {
		fmt.Println("Error writing to log file:", err)
	}
}

// 独立的错误日志记录函数
func ErrInterfaceToFile(level, message interface{}) {
	filename := time.Now().Format("2006-01-02") + "-error.log"
	filepath := logPath + "/" + filename

	file, err := os.OpenFile(filepath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Error opening log file:", err)
		return
	}
	defer file.Close()

	jsonData, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshaling data for log: %s", err)
		return
	}

	logEntry := fmt.Sprintf("[%s] %s: %s\n", time.Now().Format("2006-01-02T15:04:05"), level, string(jsonData))
	if _, err := file.WriteString(logEntry); err != nil {
		fmt.Println("Error writing to log file:", err)
	}
}
