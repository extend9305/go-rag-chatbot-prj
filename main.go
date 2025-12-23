package main

import (
	"context"
	"log"
	"time"

	"example.com/hello/chat"
	"example.com/hello/config"
	"example.com/hello/embedding"
	"example.com/hello/handler"
	"example.com/hello/reranker"
	"example.com/hello/vector"

	"github.com/gin-gonic/gin"
)

func main() {
	// .env 로드
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}
	// embedding api
	// Embedding Service 생성
	embService := embedding.NewService(cfg.EmbeddingAPIURL, cfg.EmbeddingModel)
	log.Printf("✅ Embedding service initialized (URL: %s, Model: %s)\n", cfg.EmbeddingAPIURL, cfg.EmbeddingModel)

	// reranker api
	// Reranker Service 생성
	rerankerService := reranker.NewService(cfg.RerankerAPIURL, cfg.RerankerModel)
	log.Printf("✅ Reranker service initialized (URL: %s, Model: %s)\n", cfg.RerankerAPIURL, cfg.RerankerModel)

	// llm chat api
	// llm chat Service 생성
	llmChatService := chat.NewService(cfg.LLMChatAPIURL, cfg.LLMChatModel)
	log.Printf("✅ LLM Chat service initialized (URL: %s, Model: %s)\n", cfg.LLMChatAPIURL, cfg.LLMChatModel)

	// DB 연결
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := vector.New(ctx, cfg.GetDSN())
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	log.Println("✅ Database connected successfully")

	// Handler 생성
	docHandler := handler.NewDocumentHandler(db, embService, rerankerService, llmChatService)

	// Gin 라우터
	router := gin.Default()

	// API 라우트
	api := router.Group("/api/v1")
	{
		documents := api.Group("/documents")
		{
			documents.POST("", docHandler.InsertDocument)
			documents.POST("/all", docHandler.InsertAllDocument)
			documents.GET("/:id", docHandler.GetDocument)
			documents.POST("/chat", docHandler.RagChatting)
		}
	}

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// 서버 시작
	log.Println("🚀 Server starting on :8080")
	if err := router.Run(":8080"); err != nil {
		log.Fatal("Failed to start server:", err)
	}

}
