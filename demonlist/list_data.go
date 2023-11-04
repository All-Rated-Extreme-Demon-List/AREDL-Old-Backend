package demonlist

func Aredl() ListData {
	return ListData{
		Name:                 "aredl",
		LeaderboardTableName: "aredl_leaderboard",
		SubmissionsTableName: "record_submissions",
		RecordsTableName:     "records",
		LevelTableName:       "aredl",
		CreatorTableName:     "creators",
		HistoryTableName:     "position_history",
		PointLookupTableName: "points",
		Packs: PackData{
			PackTableName:           "packs",
			PackLevelTableName:      "pack_levels",
			CompletedPacksTableName: "completed_packs",
			PackMultiplier:          0.5,
		},
	}
}

type PackData struct {
	PackTableName           string
	PackLevelTableName      string
	CompletedPacksTableName string
	PackMultiplier          float64
}

type ListData struct {
	Name                 string
	LeaderboardTableName string
	SubmissionsTableName string
	RecordsTableName     string
	LevelTableName       string
	CreatorTableName     string
	HistoryTableName     string
	PointLookupTableName string
	Packs                PackData
}
