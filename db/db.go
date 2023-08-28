package db

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

type DatabaseHandler struct {
	Conn *sql.DB
}

func (db *DatabaseHandler) InsertMessage(ctx context.Context, from string, message string) int {
	var messageID int
	err := db.Conn.QueryRowContext(ctx, "INSERT INTO messages (\"from\", content) VALUES ($1, $2) RETURNING Id", from, message).Scan(&messageID)
	if err != nil {
		fmt.Println("Failed to insert message: %v", err)
		return -1
	}

	return messageID
}

type Message struct {
	From    string
	Content string
}

func (db *DatabaseHandler) GetAllMessages(ctx context.Context) ([]Message, error) {
	query := `SELECT "from", content FROM messages`
	rows, err := db.Conn.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			var msg Message
			if err := rows.Scan(&msg.From, &msg.Content); err != nil {
				return nil, err
			}
			messages = append(messages, msg)
		}
	}

	// Check for any errors encountered during iteration
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return messages, nil
}

func NewDatabaseHandler() *DatabaseHandler {
	connStr := "user=dbenq password=test123 dbname=chat sslmode=disable host=localhost port=5432"

	// Connect to the database
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Failed to connect to the database: %v", err)
	}

	// Ping the database to ensure the connection is established
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping the database: %v", err)
	}

	fmt.Println("Successfully connected to the database!")

	return &DatabaseHandler{
		Conn: db,
	}
}
