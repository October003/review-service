package biz

// ListReviewParam 评价列表参数
type ListReviewParam struct {
	UserID int64
	Offset int
	Size   int
}

// ReplyParam 商家回复评价的参数
type ReplyReviewParam struct {
	ReviewID  int64
	StoreID   int64
	Content   string
	PicInfo   string
	VideoInfo string
}

// AppealParam 商家申诉的评价参数
type AppealReviewParam struct {
	ReviewID  int64
	StoreID   int64
	Reason    string
	Content   string
	PicInfo   string
	VideoInfo string
}

// AuditParam 运营审核评价的参数
type AuditReviewParam struct {
	ReviewID  int64
	OpUser    string
	OpReason  string
	OpRemarks string
	Status    int32
}

// AuditAppealParam 运营审核商家申诉的参数
type AuditAppealParam struct {
	AppealID  int64
	ReviewID  int64
	Status    int32
	OpUser    string
	OpRemarks string
}
