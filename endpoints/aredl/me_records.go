package aredl

import (
	"AREDL/demonlist"
	"AREDL/middlewares"
	"AREDL/util"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/daos"
	"github.com/pocketbase/pocketbase/models"
	"github.com/pocketbase/pocketbase/tools/types"
	"net/http"
)

type Record struct {
	Id      string         `db:"id" json:"id,omitempty"`
	Created types.DateTime `db:"created" json:"created,omitempty"`
	Updated types.DateTime `db:"updated" json:"updated,omitempty"`
	Level   struct {
		Id      string `db:"id" json:"id,omitempty"`
		Name    string `db:"name" json:"name,omitempty"`
		LevelId int    `db:"level_id" json:"level_id,omitempty"`
	} `db:"level" json:"level,omitempty" extend:"level,levels,id"`
	VideoUrl   string `db:"video_url" json:"video_url,omitempty"`
	Mobile     bool   `db:"mobile" json:"mobile,omitempty"`
	LdmId      int    `db:"ldm_id" json:"ldm_id,omitempty"`
	RawFootage string `db:"raw_footage" json:"raw_footage,omitempty"`
}

// registerRecordList godoc
//
//	@Summary		List records
//	@Description	Lists records ordered by the time they have been updated last.
//	@Description	Requires user permission: aredl.user_record_list
//	@Tags			aredl
//	@Security		ApiKeyAuth[authorization]
//	@Schemes		http https
//	@Produce		json
//	@Success		200	{object}	[]Record
//	@Failure		400	{object}	util.ErrorResponse
//	@Failure		403	{object}	util.ErrorResponse
//	@Router			/aredl/me/records [get]
func registerRecordList(e *echo.Group, app core.App) error {
	_, err := e.AddRoute(echo.Route{
		Method: http.MethodGet,
		Path:   "/me/records",
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(app),
			middlewares.CheckBanned(),
			middlewares.RequirePermissionGroup(app, "aredl", "user_record_list"),
		},
		Handler: func(c echo.Context) error {
			aredl := demonlist.Aredl()
			err := app.Dao().RunInTransaction(func(txDao *daos.Dao) error {
				userRecord, _ := c.Get(apis.ContextAuthRecordKey).(*models.Record)
				if userRecord == nil {
					return util.NewErrorResponse(nil, "Could not load user")
				}
				var records []Record
				tables := map[string]string{
					"base":   aredl.RecordsTableName,
					"levels": aredl.LevelTableName,
				}
				err := util.LoadFromDb(app.Dao().DB(), &records, tables, func(query *dbx.SelectQuery, prefixResolver util.PrefixResolver) {
					query.Where(dbx.HashExp{prefixResolver("submitted_by"): userRecord.Id})
					query.OrderBy(prefixResolver("updated"))
				})
				if err != nil {
					return util.NewErrorResponse(err, "could not load records")
				}
				return c.JSON(200, records)
			})
			return err
		},
	})
	return err
}
