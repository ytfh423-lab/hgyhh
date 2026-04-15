package dto

type Notify struct {
	Type    string        `json:"type"`
	Title   string        `json:"title"`
	Content string        `json:"content"`
	Values  []interface{} `json:"values"`
}

const ContentValueParam = "{{value}}"

const (
	NotifyTypeQuotaExceed         = "quota_exceed"
	NotifyTypeChannelUpdate       = "channel_update"
	NotifyTypeChannelTest         = "channel_test"
	NotifyTypeFarmCropNeedsWater  = "farm_crop_needs_water"
	NotifyTypeFarmCropNearDeath   = "farm_crop_near_death"
	NotifyTypeFarmCropStolen      = "farm_crop_stolen"
	NotifyTypeRanchAnimalCleanup  = "ranch_animal_needs_cleanup"
	NotifyTypeRanchAnimalNearDeath = "ranch_animal_near_death"
	NotifyTypeSocialOfflineMessage = "social_offline_message"
)

func NewNotify(t string, title string, content string, values []interface{}) Notify {
	return Notify{
		Type:    t,
		Title:   title,
		Content: content,
		Values:  values,
	}
}
