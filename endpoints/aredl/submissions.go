package aredl

import (
	"AREDL/demonlist"
	"AREDL/middlewares"
	"AREDL/names"
	"AREDL/util"
	"net/http"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/daos"
	"github.com/pocketbase/pocketbase/tools/types"
)

type Submission struct {
	Id      string         `db:"id" json:"id,omitempty"`
	Created types.DateTime `db:"created" json:"created,omitempty"`
	Updated types.DateTime `db:"updated" json:"updated,omitempty"`
	Level   struct {
		Id      string `db:"id" json:"id,omitempty"`
		Name    string `db:"name" json:"name,omitempty"`
		LevelId int    `db:"level_id" json:"level_id,omitempty"`
	} `db:"level" json:"level,omitempty" extend:"level,levels,id"`
	VideoUrl        string `db:"video_url" json:"video_url,omitempty"`
	Mobile          bool   `db:"mobile" json:"mobile,omitempty"`
	LdmId           int    `db:"ldm_id" json:"ldm_id,omitempty"`
	Rejected        bool   `db:"rejected" json:"rejected"`
	IsUpdate        bool   `db:"is_update" json:"is_update"`
	RawFootage      string `db:"raw_footage" json:"raw_footage,omitempty"`
	AdditionalNotes string `db:"additional_notes" json:"additional_notes"`
	Reviewer        *struct {
		Id         string `db:"id" json:"id"`
		GlobalName string `db:"global_name" json:"global_name"`
	} `db:"reviewer" json:"reviewer,omitempty" extend:"reviewer,users,id"`
	SubmittedBy struct {
		Id         string `db:"id" json:"id"`
		GlobalName string `db:"global_name" json:"global_name"`
	} `db:"submitted_by" json:"submitted_by" extend:"submitted_by,users,id"`
	Priority bool `db:"priority" json:"priority"`
}

// registerSubmissionList godoc
//
//	@Summary		List submissions
//	@Description	Lists submissions ordered by the time they have been updated last.
//	@Description	Requires user permission: aredl.submission_review
//	@Tags			aredl
//	@Param			include_rejected	query	bool	false	"include rejected submissions" default(false)
//	@Security		ApiKeyAuth[authorization]
//	@Schemes		http https
//	@Produce		json
//	@Success		200	{object}	[]Submission
//	@Failure		400	{object}	util.ErrorResponse
//	@Failure		403	{object}	util.ErrorResponse
//	@Router			/aredl/submissions [get]
func registerSubmissionList(e *echo.Group, app core.App) error {
	_, err := e.AddRoute(echo.Route{
		Method: http.MethodGet,
		Path:   "/submissions",
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(app),
			middlewares.CheckBanned(),
			middlewares.RequirePermissionGroup(app, "aredl", "submission_review"),
			middlewares.LoadParam(middlewares.LoadData{
				"include_rejected": middlewares.AddDefault(false, middlewares.LoadBool(false)),
			}),
		},
		Handler: func(c echo.Context) error {
			aredl := demonlist.Aredl()
			err := app.Dao().RunInTransaction(func(txDao *daos.Dao) error {
				var submissions []Submission
				tables := map[string]string{
					"base":   aredl.SubmissionsTableName,
					"levels": aredl.LevelTableName,
					"users":  names.TableUsers,
				}
				err := util.LoadFromDb(app.Dao().DB(), &submissions, tables, func(query *dbx.SelectQuery, prefixResolver util.PrefixResolver) {
					if !c.Get("include_rejected").(bool) {
						query.Where(dbx.HashExp{"rejected": false})
					}
					query.OrderBy(prefixResolver("updated"))
				})
				if err != nil {
					return util.NewErrorResponse(err, "could not load submissions")
				}
				return c.JSON(200, submissions)
			})
			return err
		},
	})
	return err
}
