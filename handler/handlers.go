package handler

import (
	"fmt"
	"net/http"

	"example.com/hello/chat"
	"example.com/hello/embedding"
	"example.com/hello/reranker"
	database "example.com/hello/vector"
	"github.com/gin-gonic/gin"
)

type DocumentHandler struct {
	db              *database.VectorDB
	embService      *embedding.Service
	rerankerService *reranker.Service
	llmChatService  *chat.Service
}

func NewDocumentHandler(db *database.VectorDB, embService *embedding.Service, rerankerService *reranker.Service, llmChatService *chat.Service) *DocumentHandler {
	return &DocumentHandler{
		db:              db,
		embService:      embService,
		rerankerService: rerankerService,
		llmChatService:  llmChatService,
	}
}

// RagChatting
func (h *DocumentHandler) RagChatting(c *gin.Context) {
	var req struct {
		Content string `json:"content" binding:"required"`
		//Embedding []float32 `json:"embedding" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// chatting request embedding 처리
	// embedding api로 질의문 vector 데이터로 변환
	embChatData, err := h.embService.GenerateEmbedding(c.Request.Context(), req.Content)
	fmt.Println("logging embed data:", embChatData)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}
	// vector 데이터로 db 데이터 조회
	similar, err := h.db.SearchSimilar(c.Request.Context(), embChatData, 3)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}
	fmt.Println("similar:", similar)
	// rerank 처리
	//rerank, err := h.rerankerService.FastRerank(c.Request.Context(), req.Content, similar)

	rerank, err := h.rerankerService.Rerank(c.Request.Context(), req.Content, similar)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}
	fmt.Println("rerank:", rerank)
	// llm 처리
	answer, err := h.llmChatService.Chat(c.Request.Context(), req.Content, rerank)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}

	c.JSON(http.StatusOK, gin.H{
		"answer": answer,
	})

}

// InsertDocument handles POST /documents
func (h *DocumentHandler) InsertAllDocument(c *gin.Context) {
	var req struct {
		Content []string `json:"content" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	for _, content := range req.Content {
		embeddingData, err2 := h.embService.GenerateEmbedding(c.Request.Context(), content)
		if err2 != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err2.Error()})
		}
		_, err := h.db.InsertDocument(c.Request.Context(), content, embeddingData)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Document inserted successfully",
	})
}

// InsertDocument handles POST /documents
func (h *DocumentHandler) InsertDocument(c *gin.Context) {
	var req struct {
		Content string `json:"content" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	embedding, err2 := h.embService.GenerateEmbedding(c.Request.Context(), req.Content)
	if err2 != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err2.Error()})
	}

	id, err := h.db.InsertDocument(c.Request.Context(), req.Content, embedding)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":      id,
		"message": "Document inserted successfully",
	})

}

// GetDocument handles GET /documents/:id
func (h *DocumentHandler) GetDocument(c *gin.Context) {
	var id struct {
		ID int `uri:"id" binding:"required"`
	}

	if err := c.ShouldBindUri(&id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid document ID"})
		return
	}

	doc, err := h.db.GetDocumentByID(c.Request.Context(), id.ID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Document not found"})
		return
	}

	c.JSON(http.StatusOK, doc)
}

//// SearchSimilar handles POST /documents/search
//func (h *DocumentHandler) SearchSimilar(c *gin.Context) {
//	var req models.SearchRequest
//	if err := c.ShouldBindJSON(&req); err != nil {
//		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
//		return
//	}
//
//	if req.Limit <= 0 {
//		req.Limit = 10
//	}
//
//	documents, err := h.db.SearchSimilar(c.Request.Context(), req.QueryVector, req.Limit)
//	if err != nil {
//		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
//		return
//	}
//
//	c.JSON(http.StatusOK, gin.H{
//		"results": documents,
//		"count":   len(documents),
//	})
//}
//
//// GetDocument handles GET /documents/:id
//func (h *DocumentHandler) GetDocument(c *gin.Context) {
//	id, err := strconv.Atoi(c.Param("id"))
//	if err != nil {
//		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid document ID"})
//		return
//	}
//
//	doc, err := h.db.GetDocumentByID(c.Request.Context(), id)
//	if err != nil {
//		c.JSON(http.StatusNotFound, gin.H{"error": "Document not found"})
//		return
//	}
//
//	c.JSON(http.StatusOK, doc)
//}
//
//// DeleteDocument handles DELETE /documents/:id
//func (h *DocumentHandler) DeleteDocument(c *gin.Context) {
//	id, err := strconv.Atoi(c.Param("id"))
//	if err != nil {
//		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid document ID"})
//		return
//	}
//
//	err = h.db.DeleteDocument(c.Request.Context(), id)
//	if err != nil {
//		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
//		return
//	}
//
//	c.JSON(http.StatusOK, gin.H{"message": "Document deleted successfully"})
//}
//
//// GetStats handles GET /documents/stats
//func (h *DocumentHandler) GetStats(c *gin.Context) {
//	count, err := h.db.GetDocumentCount(c.Request.Context())
//	if err != nil {
//		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
//		return
//	}
//
//	c.JSON(http.StatusOK, gin.H{
//		"total_documents": count,
//	})
//}
