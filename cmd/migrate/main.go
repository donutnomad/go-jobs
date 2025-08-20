package main

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	// 连接数据库
	dsn := "root:123456@tcp(127.0.0.1:3306)/jobs?charset=utf8mb4&parseTime=True&loc=Local&multiStatements=true"
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// 读取 SQL 文件
	sqlBytes, err := ioutil.ReadFile("scripts/migrate_task_types.sql")
	if err != nil {
		log.Fatal("Failed to read SQL file:", err)
	}

	// 分割 SQL 语句并执行
	sqlStatements := strings.Split(string(sqlBytes), ";")
	for _, stmt := range sqlStatements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" || strings.HasPrefix(stmt, "--") {
			continue
		}

		fmt.Printf("Executing: %s...\n", stmt[:min(50, len(stmt))])
		_, err := db.Exec(stmt)
		if err != nil {
			log.Printf("Error executing statement: %v", err)
		}
	}

	fmt.Println("Migration completed successfully!")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
