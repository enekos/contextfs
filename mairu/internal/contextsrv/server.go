package contextsrv

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	engine *gin.Engine
}

func NewHandler(svc Service, authToken string) *Handler {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(func(c *gin.Context) {
		if authToken == "" {
			c.Next()
			return
		}
		if c.GetHeader("X-Context-Token") != authToken {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		c.Next()
	})

	api := r.Group("/api")
	{
		api.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, svc.Health())
		})

		api.GET("/cluster", func(c *gin.Context) {
			c.JSON(http.StatusOK, svc.ClusterStats())
		})

		api.GET("/dashboard", func(c *gin.Context) {
			limit := intParam(c.Query("limit"), 200)
			out, err := svc.Dashboard(limit, c.Query("project"))
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, out)
		})

		api.POST("/memories", func(c *gin.Context) {
			var req struct {
				Project    string `json:"project"`
				Content    string `json:"content"`
				Category   string `json:"category"`
				Owner      string `json:"owner"`
				Importance int    `json:"importance"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
				return
			}
			out, err := svc.CreateMemory(MemoryCreateInput{
				Project:    req.Project,
				Content:    req.Content,
				Category:   req.Category,
				Owner:      req.Owner,
				Importance: req.Importance,
			})
			if err != nil {
				if errors.Is(err, ErrModerationRejected) {
					c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
					return
				}
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusCreated, out)
		})

		api.GET("/memories", func(c *gin.Context) {
			limit := intParam(c.Query("limit"), 200)
			items, err := svc.ListMemories(c.Query("project"), limit)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, items)
		})

		api.PUT("/memories", func(c *gin.Context) {
			var req struct {
				ID         string `json:"id"`
				Content    string `json:"content"`
				Category   string `json:"category"`
				Owner      string `json:"owner"`
				Importance int    `json:"importance"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
				return
			}
			out, err := svc.UpdateMemory(MemoryUpdateInput{
				ID:         req.ID,
				Content:    req.Content,
				Category:   req.Category,
				Owner:      req.Owner,
				Importance: req.Importance,
			})
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, out)
		})

		api.DELETE("/memories", func(c *gin.Context) {
			if err := svc.DeleteMemory(c.Query("id")); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

		api.POST("/skills", func(c *gin.Context) {
			var req struct {
				Project     string `json:"project"`
				Name        string `json:"name"`
				Description string `json:"description"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
				return
			}
			out, err := svc.CreateSkill(SkillCreateInput{
				Project:     req.Project,
				Name:        req.Name,
				Description: req.Description,
			})
			if err != nil {
				if errors.Is(err, ErrModerationRejected) {
					c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
					return
				}
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusCreated, out)
		})

		api.GET("/skills", func(c *gin.Context) {
			limit := intParam(c.Query("limit"), 200)
			items, err := svc.ListSkills(c.Query("project"), limit)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, items)
		})

		api.PUT("/skills", func(c *gin.Context) {
			var req struct {
				ID          string `json:"id"`
				Name        string `json:"name"`
				Description string `json:"description"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
				return
			}
			out, err := svc.UpdateSkill(SkillUpdateInput{
				ID:          req.ID,
				Name:        req.Name,
				Description: req.Description,
			})
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, out)
		})

		api.DELETE("/skills", func(c *gin.Context) {
			if err := svc.DeleteSkill(c.Query("id")); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

		api.POST("/context", func(c *gin.Context) {
			var req struct {
				URI       string  `json:"uri"`
				Project   string  `json:"project"`
				ParentURI *string `json:"parent_uri"`
				Name      string  `json:"name"`
				Abstract  string  `json:"abstract"`
				Overview  string  `json:"overview"`
				Content   string  `json:"content"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
				return
			}
			out, err := svc.CreateContextNode(ContextCreateInput{
				URI:       req.URI,
				Project:   req.Project,
				ParentURI: req.ParentURI,
				Name:      req.Name,
				Abstract:  req.Abstract,
				Overview:  req.Overview,
				Content:   req.Content,
			})
			if err != nil {
				if errors.Is(err, ErrModerationRejected) {
					c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
					return
				}
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusCreated, out)
		})

		api.GET("/context", func(c *gin.Context) {
			limit := intParam(c.Query("limit"), 200)
			var parentURI *string
			if v := c.Query("parentUri"); v != "" {
				parentURI = &v
			}
			items, err := svc.ListContextNodes(c.Query("project"), parentURI, limit)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, items)
		})

		api.PUT("/context", func(c *gin.Context) {
			var req struct {
				URI      string `json:"uri"`
				Name     string `json:"name"`
				Abstract string `json:"abstract"`
				Overview string `json:"overview"`
				Content  string `json:"content"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
				return
			}
			out, err := svc.UpdateContextNode(ContextUpdateInput{
				URI:      req.URI,
				Name:     req.Name,
				Abstract: req.Abstract,
				Overview: req.Overview,
				Content:  req.Content,
			})
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, out)
		})

		api.DELETE("/context", func(c *gin.Context) {
			if err := svc.DeleteContextNode(c.Query("uri")); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

		api.GET("/search", func(c *gin.Context) {
			query := c.Query("q")
			if query == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "q parameter required"})
				return
			}
			topK := intParam(c.Query("topK"), 10)
			store := c.DefaultQuery("type", "all")
			out, err := svc.Search(SearchOptions{
				Query:         query,
				Project:       c.Query("project"),
				Store:         store,
				TopK:          topK,
				MinScore:      floatParam(c.Query("minScore"), 0),
				Highlight:     c.Query("highlight") == "true",
				Fuzziness:     c.Query("fuzziness"),
				PhraseBoost:   floatParam(c.Query("phraseBoost"), 0),
				WeightVector:  floatParam(c.Query("weightVector"), 0),
				WeightKeyword: floatParam(c.Query("weightKeyword"), 0),
				WeightRecency: floatParam(c.Query("weightRecency"), 0),
				WeightImp:     floatParam(c.Query("weightImportance"), 0),
				RecencyScale:  c.Query("recencyScale"),
				RecencyDecay:  floatParam(c.Query("recencyDecay"), 0),
			})
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, out)
		})

		api.POST("/vibe/query", func(c *gin.Context) {
			var req struct {
				Prompt  string `json:"prompt"`
				Project string `json:"project"`
				TopK    int    `json:"topK"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
				return
			}
			out, err := svc.VibeQuery(req.Prompt, req.Project, req.TopK)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, out)
		})

		api.POST("/vibe/mutation/plan", func(c *gin.Context) {
			var req struct {
				Prompt  string `json:"prompt"`
				Project string `json:"project"`
				TopK    int    `json:"topK"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
				return
			}
			out, err := svc.PlanVibeMutation(req.Prompt, req.Project, req.TopK)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, out)
		})

		api.POST("/vibe/mutation/execute", func(c *gin.Context) {
			var req struct {
				Project    string           `json:"project"`
				Operations []VibeMutationOp `json:"operations"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
				return
			}
			results, err := svc.ExecuteVibeMutation(req.Operations, req.Project)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"results": results})
		})

		api.GET("/moderation/queue", func(c *gin.Context) {
			limit := intParam(c.Query("limit"), 100)
			items, err := svc.ListModerationQueue(limit)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"items": items})
		})

		api.POST("/moderation/review", func(c *gin.Context) {
			var req struct {
				EventID  int64  `json:"event_id"`
				Decision string `json:"decision"`
				Reviewer string `json:"reviewer"`
				Notes    string `json:"notes"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
				return
			}
			if err := svc.ReviewModeration(ModerationReviewInput{
				EventID:  req.EventID,
				Decision: req.Decision,
				Reviewer: req.Reviewer,
				Notes:    req.Notes,
			}); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})
	}

	return &Handler{engine: r}
}

func intParam(raw string, fallback int) int {
	if raw == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}

func floatParam(raw string, fallback float64) float64 {
	if raw == "" {
		return fallback
	}
	n, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return fallback
	}
	return n
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.engine.ServeHTTP(w, r)
}
