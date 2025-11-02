package types

type UploadFilters struct {
	Module     string   `form:"module"`
	EntityType string   `form:"entity_type"`
	EntityID   string   `form:"entity_id"`
	FileTypes  []string `form:"file_types"`
	Usage      string   `form:"usage"`
	IsPublic   *bool    `form:"is_public"`
	Limit      int      `form:"limit"`
	Offset     int      `form:"offset"`
}
