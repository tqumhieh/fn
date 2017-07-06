package server

import (
	"context"
	"net/http"
	"path"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"
	"gitlab-odx.oracle.com/odx/functions/api"
	"gitlab-odx.oracle.com/odx/functions/api/models"
	"gitlab-odx.oracle.com/odx/functions/api/runner/common"
)

/* handleRouteCreateOrUpdate is used to handle POST PUT and PATCH for routes.
   Post will only create route if its not there and create app if its not.
       create only
	   Post does not skip validation of zero values
   Put will create app if its not there and if route is there update if not it will create new route.
       update if exists or create if not exists
	   Put does not skip validation of zero values
   Patch will not create app if it does not exist since the route needs to exist as well...
       update only
	   Patch accepts partial updates / skips validation of zero values.
*/
func (s *Server) handleRouteCreateOrUpdate(c *gin.Context) {
	ctx := c.MustGet("ctx").(context.Context)
	log := common.Logger(ctx)
	method := strings.ToUpper(c.Request.Method)

	var wroute models.RouteWrapper

	err := s.bindAndValidate(ctx, c, log, method, &wroute)
	if err != nil {
		c.JSON(http.StatusBadRequest, simpleError(err))
		return
	}

	//Create the app if it does not exist.
	err = s.createApp(ctx, c, log, wroute, method)
	if err != nil {
		c.JSON(http.StatusInternalServerError, simpleError(err))
		return
	}

	resp, err := s.updateOrInsertRoute(ctx, method, wroute)
	if err != nil {
		handleErrorResponse(c, err)
		return
	}

	s.cacheRefresh(resp.Route)

	c.JSON(http.StatusOK, resp)
}

func (s *Server) createApp(ctx context.Context, c *gin.Context, log logrus.FieldLogger, wroute models.RouteWrapper, method string) error {
	if !(method == http.MethodPost || method == http.MethodPut) {
		return nil
	}
	var app *models.App
	var err error
	app, err = s.Datastore.GetApp(ctx, wroute.Route.AppName)
	if err != nil && err != models.ErrAppsNotFound {
		log.WithError(err).Error(models.ErrAppsGet)
		return models.ErrAppsGet
	} else if app == nil {
		// Create a new application and add the route to that new application
		newapp := &models.App{Name: wroute.Route.AppName}
		if err = newapp.Validate(); err != nil {
			log.Error(err)
			return err
		}

		err = s.FireBeforeAppCreate(ctx, newapp)
		if err != nil {
			log.WithError(err).Error(models.ErrAppsCreate)
			return models.ErrAppsCreate
		}

		_, err = s.Datastore.InsertApp(ctx, newapp)
		if err != nil {
			log.WithError(err).Error(models.ErrRoutesCreate)
			return models.ErrRoutesCreate
		}

		err = s.FireAfterAppCreate(ctx, newapp)
		if err != nil {
			log.WithError(err).Error(models.ErrRoutesCreate)
			return models.ErrRoutesCreate
		}

	}
	return nil
}

func (s *Server) bindAndValidate(ctx context.Context, c *gin.Context, log logrus.FieldLogger, method string, wroute *models.RouteWrapper) error {
	err := c.BindJSON(wroute)
	if err != nil {
		log.WithError(err).Debug(models.ErrInvalidJSON)
		return models.ErrInvalidJSON
	}

	if wroute.Route == nil {
		log.WithError(err).Debug(models.ErrRoutesMissingNew)
		return models.ErrRoutesMissingNew
	}
	// MAKE SIMPLEER TO READ / REFACTOR then help denis with the ci errors.
	wroute.Route.AppName = c.MustGet(api.AppName).(string)

	if method == http.MethodPut || method == http.MethodPatch {
		p := path.Clean(c.MustGet(api.Path).(string))

		if wroute.Route.Path != "" && wroute.Route.Path != p {
			log.Debug(models.ErrRoutesPathImmutable)
			return models.ErrRoutesPathImmutable
		}
		wroute.Route.Path = p
	}

	wroute.Route.SetDefaults()

	if err = wroute.Validate(method == http.MethodPatch); err != nil {
		log.WithError(err).Debug(models.ErrRoutesCreate)
		return err
	}
	return nil
}

func (s *Server) updateOrInsertRoute(ctx context.Context, method string, wroute models.RouteWrapper) (routeResponse, error) {
	var route *models.Route
	var err error
	resp := routeResponse{"Route successfully created", nil}
	up := routeResponse{"Route successfully updated", nil}

	switch method {
	case http.MethodPost:
		route, err = s.Datastore.InsertRoute(ctx, wroute.Route)
	case http.MethodPut:
		route, err = s.Datastore.UpdateRoute(ctx, wroute.Route)
		if err == models.ErrRoutesNotFound {
			// try insert then
			route, err = s.Datastore.InsertRoute(ctx, wroute.Route)
		}
	case http.MethodPatch:
		route, err = s.Datastore.UpdateRoute(ctx, wroute.Route)
		resp = up
	}
	resp.Route = route
	return resp, err
}
