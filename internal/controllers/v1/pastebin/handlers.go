package pastebin

import (
	"errors"
	"net/http"
	"triton-backend/internal/database"
	"triton-backend/internal/merrors"
	"triton-backend/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PastebinHandler struct {
	db *pgxpool.Pool
}

func Handler(db *pgxpool.Pool) *PastebinHandler {
	return &PastebinHandler{
		db: db,
	}
}

func (p *PastebinHandler) CreatePastebin(c *gin.Context) {
	var input struct {
		UserUUID  uuid.UUID `json:"user_uuid" binding:"required,uuid"`
		Title     string    `json:"title" binding:"required"`
		Content   string    `json:"content" binding:"required"`
		Extension string    `json:"extension" binding:"required"`
		URL       string    `json:"url" binding:"required"`
	}
	binding.EnableDecoderDisallowUnknownFields = true
	err := c.ShouldBindJSON(&input)
	if err != nil {
		merrors.Validation(c, err.Error())
		return
	}

	tx, err := p.db.Begin(c)
	if err != nil {
		merrors.InternalServer(c, err.Error())
		return
	}
	defer tx.Rollback(c)
	qtx := database.New(p.db).WithTx(tx)

	url_uuid, err := qtx.CreateURL(c, input.URL)
	var e *pgconn.PgError
	if errors.As(err, &e) && e.Code == pgerrcode.UniqueViolation {
		merrors.Validation(c, "url name already is in use, please choose another!")
		return
	} else if err != nil {
		merrors.InternalServer(c, err.Error())
		return
	}

	pastebin, err := qtx.CreatePastebin(c, database.CreatePastebinParams{
		UserUuid:  input.UserUUID,
		Title:     input.Title,
		Content:   input.Content,
		UrlUuid:   url_uuid,
		Extension: input.Extension,
	})
	if err != nil {
		merrors.InternalServer(c, err.Error())
		return
	}

	err = tx.Commit(c)
	if err != nil {
		merrors.InternalServer(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, utils.BaseResponse{
		Success:    true,
		Message:    "Pastebin successfully retreived",
		Data:       pastebin,
		StatusCode: http.StatusOK,
	})
}

func (p *PastebinHandler) GetPastebin(c *gin.Context) {
	var input struct {
		URL string `uri:"url"`
	}
	err := c.ShouldBindUri(&input)
	if err != nil {
		merrors.Validation(c, err.Error())
		return
	}

	q := database.New(p.db)
	pastebin, err := q.GetPastebin(c, input.URL)
	if errors.Is(err, pgx.ErrNoRows) {
		merrors.NotFound(c, "This page does not exist!")
		return
	} else if err != nil {
		merrors.InternalServer(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, utils.BaseResponse{
		Success:    true,
		Message:    "Pastebin successfully retrieved",
		Data:       pastebin,
		StatusCode: http.StatusOK,
	})

}
