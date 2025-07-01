package api

import (
	"net/http"
	"strconv"

	"d:\easypeek\condpeek\internal\ai"
	"d:\easypeek\condpeek\internal\services"
	"d:\easypeek\condpeek\internal\utils"
	"github.com/gin-gonic/gin"
)

// AIHandler handles AI-related API requests.
type AIHandler struct {
	aiService   *ai.AIService
	newsService *services.NewsService
}

// NewAIHandler creates a new AIHandler.
func NewAIHandler(aiService *ai.AIService, newsService *services.NewsService) *AIHandler {
	return &AIHandler{
		aiService:   aiService,
		newsService: newsService,
	}
}

// SummarizeNews godoc
// @Summary      Summarize a news article
// @Description  get a summary for a news article by its ID
// @Tags         AI
// @Accept       json
// @Produce      json
// @Param        id   path      int  true  "News ID"
// @Success      200  {object}  map[string]string
// @Failure      400  {object}  utils.ErrorResponse
// @Failure      404  {object}  utils.ErrorResponse
// @Failure      500  {object}  utils.ErrorResponse
// @Router       /news/{id}/summarize [post]
func (h *AIHandler) SummarizeNews(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, "Invalid news ID")
		return
	}

	news, err := h.newsService.GetNewsByID(c.Request.Context(), uint(id))
	if err != nil {
		utils.RespondWithError(c, http.StatusNotFound, "News not found")
		return
	}

	summary, err := h.aiService.SummarizeNews(c.Request.Context(), news)
	if err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, "Failed to generate summary")
		return
	}

	utils.RespondWithJSON(c, http.StatusOK, gin.H{"summary": summary})
}