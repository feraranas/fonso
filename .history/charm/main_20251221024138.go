package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3" // Import sqlite3 driver
)

// ///////////////////////////////////////
// ///////// DATABASE STUFF ///////////
// ///////////////////////////////////////
type DBConnection struct {
	DB   *sql.DB
	path string
}

var sqlOpen = sql.Open

var pragmaExec = func(db *sql.DB, query string) (sql.Result, error) {
	return db.Exec(query)
}

// ///////////////////////////////////////
// ///////// MODELS ///////////
// ///////////////////////////////////////
type Article struct {
	ID      int
	Title   string
	Content string
}

// ///////////////////////////////////////
// ///////// REPOSITORY ///////////
// ///////////////////////////////////////
type ArticleRepository struct {
	db *sql.DB
}

// NewArticleRepository creates a new article repository
func NewArticleRepository(db *sql.DB) *ArticleRepository {
	return &ArticleRepository{db: db}
}

// CreateTable creates the articles table if it doesn't exist
func (r *ArticleRepository) CreateTable() error {
	query := `
		CREATE TABLE IF NOT EXISTS articles (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT NOT NULL,
			content TEXT
		)
	`
	_, err := r.db.Exec(query)
	return err
}

// Create inserts a new article
func (r *ArticleRepository) Create(title, content string) (int64, error) {
	query := `INSERT INTO articles (title, content) VALUES (?, ?)`
	result, err := r.db.Exec(query, title, content)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// GetByID retrieves an article by ID
func (r *ArticleRepository) GetByID(id int) (*Article, error) {
	query := `SELECT id, title, content FROM articles WHERE id = ?`

	article := &Article{}
	err := r.db.QueryRow(query, id).Scan(&article.ID, &article.Title, &article.Content)
	if err != nil {
		return nil, err
	}
	return article, nil
}

// GetAll retrieves all articles
func (r *ArticleRepository) GetAll() ([]Article, error) {
	query := `SELECT id, title, content FROM articles`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var articles []Article
	for rows.Next() {
		var article Article
		if err := rows.Scan(&article.ID, &article.Title, &article.Content); err != nil {
			return nil, err
		}
		articles = append(articles, article)
	}

	return articles, rows.Err()
}

// Update updates an existing article
func (r *ArticleRepository) Update(id int, title, content string) error {
	query := `UPDATE articles SET title = ?, content = ? WHERE id = ?`
	_, err := r.db.Exec(query, title, content, id)
	return err
}

// Delete removes an article by ID
func (r *ArticleRepository) Delete(id int) error {
	query := `DELETE FROM articles WHERE id = ?`
	_, err := r.db.Exec(query, id)
	return err
}

type Repositories struct {
	Articles *ArticleRepository
}

func NewRepositories(db *sql.DB) *Repositories {
	return &Repositories{
		Articles: NewArticleRepository(db),
	}
}

// ///////////////////////////////////////
// ///////// MAIN ///////////
// ///////////////////////////////////////
func main() {
	dbPath := "./mock.db"

	// Open database connection
	db, err := sqlOpen("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Enable foreign keys
	if _, err := pragmaExec(db, "PRAGMA foreign_keys = ON"); err != nil {
		log.Fatalf("Failed to enable foreign keys: %v", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	database := &DBConnection{DB: db, path: dbPath}
	repos := NewRepositories(database.DB)

	// Create table
	if err := repos.Articles.CreateTable(); err != nil {
		log.Fatalf("Failed to create table: %v", err)
	}

	// Example: Create a new article
	id, err := repos.Articles.Create("My First Article", "This is the content of my first article")
	if err != nil {
		log.Fatalf("Failed to create article: %v", err)
	}
	fmt.Printf("Created article with ID: %d\n", id)

	// Example: Get article by ID
	article, err := repos.Articles.GetByID(int(id))
	if err != nil {
		log.Fatalf("Failed to get article: %v", err)
	}
	fmt.Printf("Retrieved: ID=%d, Title=%s, Content=%s\n", article.ID, article.Title, article.Content)

	// Example: Get all articles
	articles, err := repos.Articles.GetAll()
	if err != nil {
		log.Fatalf("Failed to get all articles: %v", err)
	}
	fmt.Printf("Total articles: %d\n", len(articles))

	// Example: Update article
	if err := repos.Articles.Update(int(id), "Updated Title", "Updated content"); err != nil {
		log.Fatalf("Failed to update article: %v", err)
	}
	fmt.Println("Article updated successfully")

	// Example: Delete article (commented out to keep data)
	// if err := repos.Articles.Delete(int(id)); err != nil {
	// 	log.Fatalf("Failed to delete article: %v", err)
	// }
	// fmt.Println("Article deleted successfully")
}
