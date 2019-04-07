package main

import (
	"context"
	"database/sql"
	"fmt"
	"lightning-poll/votes"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"lightning-poll/lnd"
	"lightning-poll/polls"
	"lightning-poll/types"
)

type Env struct {
	db  *sql.DB
	lnd lnd.Client
}

func (e Env) GetDB() *sql.DB {
	return e.db
}

func (e Env) GetLND() lnd.Client {
	return e.lnd
}

func initializeRoutes(e Env) {
	router.GET("/", e.showHomePage)
	router.GET("/create", e.createPollPage)
	router.GET("/view/:id", e.viewPollPage)
	router.GET("/vote/:id", e.viewVotePage)

	router.POST("/create", e.createPollPost)
	router.POST("/vote", e.createVotePost)
}

func (e *Env) showHomePage(c *gin.Context) {
	polls, err := polls.ListActivePolls(context.Background(), e)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
	}

	c.HTML(
		http.StatusOK,
		"home.html",
		gin.H{
			"title": "Lightning Poll - Home",
			"polls": polls,
		},
	)

}

func (e *Env) createPollPage(c *gin.Context) {
	c.HTML(
		http.StatusOK,
		"create.html",
		gin.H{
			"title": "Lightning Poll - Create",
		},
	)
}

func (e *Env) viewPollPage(c *gin.Context) {
	id := getInt(c, "id")

	poll, err := polls.LookupPoll(context.Background(), e, id)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
	}

	c.HTML(
		http.StatusOK,
		"view.html",
		gin.H{
			"title": "Lightning Poll -View Poll",
			"poll":  poll,
		},
	)
}

func (e *Env) viewVotePage(c *gin.Context) {
	id := getInt(c, "id")

	vote, err := votes.Lookup(c.Request.Context(), e, id)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
	}

	poll, err := polls.LookupPoll(c.Request.Context(), e, vote.PollID)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
	}

	c.HTML(
		http.StatusOK,
		"vote.html",
		gin.H{
			"title": "Lightning Poll - View Vote",
			"poll":  poll,
			"vote":  vote,
		},
	)
}

func getInt(c *gin.Context, field string) int64 {
	str, ok := c.Params.Get(field)
	if !ok {
		c.AbortWithStatus(http.StatusInternalServerError)
	}

	num, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
	}
	return num
}

func getPostInt(c *gin.Context, field string) int64 {
	num, err := strconv.ParseInt(c.PostForm(field), 10, 64)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
	}
	return num
}

func (e *Env) createPollPost(c *gin.Context) {
	question := c.PostForm("question")
	payReq := c.PostForm("invoice")
	sats := getPostInt(c, "satoshis")

	expiry := getPostInt(c, "expiry")
	expirySeconds := expiry * 60 * 60 // hours to seconds

	options := c.PostForm("added[]")

	id, err := polls.CreatePoll(context.Background(), e, question, payReq,
		types.RepaySchemeMajority, strings.Split(options, ","), expirySeconds, sats, 0)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
	}

	// TODO(carla): figure out non hacky redirect
	c.Params = append(c.Params, gin.Param{Key: "id", Value: fmt.Sprintf("%v", id)})
	e.viewPollPage(c)
}

func (e *Env) createVotePost(c *gin.Context) {
	pollID := getPostInt(c, "poll_id")
	optionID := getPostInt(c, "option_id")
	expiry := getPostInt(c, "expiry")
	expirySeconds := expiry * 60 * 60 // hours to seconds

	id, err := votes.Create(c.Request.Context(), e, pollID, optionID, expirySeconds, expiry)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
	}

	//TODO(carla): stop being hacky
	c.Params = append(c.Params, gin.Param{Key: "id", Value: fmt.Sprintf("%v", id)})
	e.viewVotePage(c)
}
