package vector

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"
)

type VectorDB struct {
	pool *pgxpool.Pool
}

// New creates a new VectorDB instance
func New(ctx context.Context, connString string) (*VectorDB, error) {
	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// 연결 테스트
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	db := &VectorDB{pool: pool}

	return db, nil
}

// Close closes the database connection
func (db *VectorDB) Close() {
	db.pool.Close()
}

// InsertDocument inserts a document with embedding
func (db *VectorDB) InsertDocument(ctx context.Context, content string, embedding []float32) (int, error) {
	if len(embedding) != 1024 {
		return 0, fmt.Errorf("embedding must be 1024 dimensions, got %d", len(embedding))
	}

	var id int
	err := db.pool.QueryRow(ctx, `
        INSERT INTO documents (content, embedding)
        VALUES ($1, $2)
        RETURNING id
    `, content, pgvector.NewVector(embedding)).Scan(&id)

	if err != nil {
		return 0, fmt.Errorf("failed to insert document: %w", err)
	}

	return id, nil
}

// SearchSimilar searches for similar documents
func (db *VectorDB) SearchSimilar(ctx context.Context, queryVector []float32, limit int) ([]Document, error) {
	if len(queryVector) != 1024 {
		return nil, fmt.Errorf("query vector must be 1024 dimensions, got %d", len(queryVector))
	}

	query := `
        SELECT id, content, embedding <=> $1 AS distance
        FROM documents
        ORDER BY embedding <=> $1
        LIMIT $2
    `
	vec := pgvector.NewVector(queryVector)
	// 🔍 디버깅: 벡터 첫 10개 값 출력
	log.Printf("🔍 Query vector (first 10): %v...", queryVector[:10])
	log.Printf("🔍 Converted pgvector: %v", vec)
	log.Printf("🔍 Query: %s", query)
	log.Printf("🔍 Limit: %d", limit)

	// 쿼리 로그 출력
	log.Printf("Executing query: %s\nParams: vector(len=%d), limit=%d", query, len(queryVector), limit)

	rows, err := db.pool.Query(ctx, query, vec, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search: %w", err)
	}
	defer rows.Close()

	var documents []Document
	for rows.Next() {
		var doc Document
		err := rows.Scan(&doc.ID, &doc.Content, &doc.Distance)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		documents = append(documents, doc)
	}

	return documents, rows.Err()
}

// GetDocumentByID retrieves a document by ID
func (db *VectorDB) GetDocumentByID(ctx context.Context, id int) (*Document, error) {
	var doc Document
	var embedding pgvector.Vector

	err := db.pool.QueryRow(ctx, `
        SELECT id, content, embedding, metadata, created_at
        FROM documents
        WHERE id = $1
    `, id).Scan(&doc.ID, &doc.Content, &embedding)

	if err != nil {
		return nil, fmt.Errorf("failed to get document: %w", err)
	}

	doc.Embedding = embedding.Slice()
	return &doc, nil
}

// DeleteDocument deletes a document by ID
func (db *VectorDB) DeleteDocument(ctx context.Context, id int) error {
	result, err := db.pool.Exec(ctx, "DELETE FROM documents WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("failed to delete document: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("document not found")
	}

	return nil
}

// GetDocumentCount returns the total number of documents
func (db *VectorDB) GetDocumentCount(ctx context.Context) (int, error) {
	var count int
	err := db.pool.QueryRow(ctx, "SELECT COUNT(*) FROM documents").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count documents: %w", err)
	}
	return count, nil
}

// Document represents a document with embedding
type Document struct {
	ID        int       `json:"id"`
	Content   string    `json:"content"`
	Embedding []float32 `json:"embedding,omitempty"`
	Distance  float64   `json:"distance,omitempty"`
}
