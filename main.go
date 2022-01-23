package main

import (
	"bgweb-api/api"
	"bgweb-api/docs"
	"bgweb-api/gnubg"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title BGWeb API
// @version 1.0
// @description BGWeb API
// @termsOfService /terms

// @license.name MIT
// @license.url /license

// @BasePath /api/v1
// @schemes http
func main() {
	var dataDir = readEnv("BGWEB_DATADIR", "./data")
	var port = readEnv("BGWEB_PORT", "8080")

	if err := gnubg.Init(dataDir); err != nil {
		panic(err)
	}

	r := gin.Default()
	docs.SwaggerInfo.BasePath = "/api/v1"
	v1 := r.Group("/api/v1")
	{
		v1.POST("/getmoves", getMoves)
	}
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))
	r.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/swagger/index.html")
	})

	r.Run(":" + port)
}

// getMoves godoc
// @Summary Get moves for a given board layout and dice roll
// @Tags GameAnalysis
// @Accept json
// @Produce json
// @Param        args   body      api.MoveArgs  false  "Move arguments"
// @Success 200 {object} []api.Move
// @Router /getmoves [post]
func getMoves(c *gin.Context) {
	var args = api.MoveArgs{
		Player:     "x",
		MaxMoves:   0,
		ScoreMoves: true,
		Cubeful:    false,
	}

	if err := c.BindJSON(&args); err != nil {
		c.JSON(http.StatusBadRequest, err)
		return
	}

	moves, err := api.GetMoves(args)

	if err != nil {
		c.JSON(http.StatusBadRequest, err)
		return
	}

	c.JSON(http.StatusOK, moves)
}

func readEnv(name string, defaultValue string) string {
	var env = os.Getenv(name)
	if len(env) > 0 {
		return env
	}
	return defaultValue
}
