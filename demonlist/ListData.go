package demonlist

func Aredl() ListData {
	return ListData{
		LeaderboardTableName: "aredl_leaderboard",
		SubmissionTableName:  "record_submissions",
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
	LeaderboardTableName string
	SubmissionTableName  string
	LevelTableName       string
	CreatorTableName     string
	HistoryTableName     string
	PointLookupTableName string
	Packs                PackData
}
