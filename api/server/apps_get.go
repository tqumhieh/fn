package server

import (
	"net/http"

	"github.com/fnproject/fn/api"
	"github.com/fnproject/fn/api/models"
	"github.com/gin-gonic/gin"
)

func (s *Server) handleAppGet(c *gin.Context) {
	ctx := c.Request.Context()

	app := &models.App{Name: c.MustGet(api.App).(string)}
	err := s.FireBeforeAppGet(ctx, app)
	if err != nil {
		handleErrorResponse(c, err)
		return
	}

	app, err = s.datastore.GetAppByName(ctx, app.Name)
	if err != nil {
		handleErrorResponse(c, err)
		return
	}

	err = s.FireAfterAppGet(ctx, app)
	if err != nil {
		handleErrorResponse(c, err)
		return
	}

	c.JSON(http.StatusOK, appResponse{"Successfully loaded app", app})
}
